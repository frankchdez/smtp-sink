package main

import (
	"encoding/json"
	"os"
	"os/user"
	"strings"
	"sync"

	"github.com/flashmob/go-guerrilla/backends"
	"github.com/flashmob/go-guerrilla/mail"
	"github.com/flashmob/go-guerrilla/response"
)

// Config maps to the backends config of the Go-guerrilla
type Config struct {
	// This is a more advance option of user - dir mapping
	/*
		Example:
		...
			"user_dirs_map": [
				{
					"email": "user1",
					"path": "/var/mail/dir1",
				},
				{
					"email": "user2",
					"path": "/var/mail/dir2",
				},
			]
		...
	*/
	Dirs string `json:"user_dirs_map"`
}

// DirConfig maps emails to directory path
type DirConfig struct {
	Email string `json:"email"`
	Path  string `json:"path"`
}

type processor struct {
	userMap map[string]*DirConfig
	config  *Config
}

var (
	homeDir     string
	homeDirErr  error
	homeDirInit sync.Once
)

func newProcessor(config *Config) (*processor, error) {
	p := &processor{}
	p.config = config

	var dirs []DirConfig
	if err := json.Unmarshal([]byte(p.config.Dirs), &dirs); err != nil {
		return nil, err
	}

	p.userMap = make(map[string]*DirConfig, 0)
	for i := range dirs {
		d := dirs[i].Path

		if strings.Index(d, "~/") == 0 {
			// expand the ~/ to home dir
			homeDirInit.Do(func() {
				usr, err := user.Current()
				if err != nil {
					backends.Log().WithError(err).Error("could not expand ~/ to homedir")
					homeDirErr = err
				}
				homeDir = usr.HomeDir
			})

			if homeDirErr != nil {
				return nil, homeDirErr
			}

			d = homeDir + d[1:]
		}

		if _, err := os.Stat(d); err != nil {
			return nil, err
		}

		dirs[i].Path = d
		p.userMap[dirs[i].Email] = &dirs[i]
	}

	return p, nil
}

func (p *processor) validateRcpt(addr *mail.Address) backends.RcptError {
	u := strings.ToLower(addr.User)
	if _, found := p.userMap[u]; !found {
		return backends.NoSuchUser
	}
	return nil
}

var decorator = func() backends.Decorator {
	// config will be populated by the initFunc
	var p *processor

	// The following initialization is run when the program first starts

	// initFunc is an initializer function which is called when our processor gets created.
	// It gets called for every worker
	initializer := backends.InitializeWith(func(backendConfig backends.BackendConfig) error {
		configType := backends.BaseConfig(&Config{})
		bcfg, err := backends.Svc.ExtractConfig(backendConfig, configType)

		if err != nil {
			return err
		}

		c := bcfg.(*Config)
		if p, err = newProcessor(c); err != nil {
			return err
		}

		return nil
	})
	// register our initializer
	backends.Svc.AddInitializer(initializer)

	return func(c backends.Processor) backends.Processor {
		// The function will be called on each email transaction.
		// On success, it forwards to the next step in the processor call-stack,
		// or returns with an error if failed
		return backends.ProcessWith(func(e *mail.Envelope, task backends.SelectTask) (backends.Result, error) {
			switch task {
			case backends.TaskValidateRcpt:
				// Check the recipients for each RCPT command.
				// This is called each time a recipient is added,
				// validate only the "last" recipient that was appended
				if size := len(e.RcptTo); size > 0 {
					if err := p.validateRcpt(&e.RcptTo[size-1]); err != nil {
						backends.Log().WithError(err).Info("recipient not configured: ", e.RcptTo[size-1].User)
						return backends.NewResult(response.Canned.FailRcptCmd), err
					}
				}
				return c.Process(e, task)
			case backends.TaskSaveMail:
				for i := range e.RcptTo {
					u := strings.ToLower(e.RcptTo[i].User)

					dirConfig, ok := p.userMap[u]
					if !ok {
						// no such user
						continue
					}

					if err := parseMail(e.NewReader(), dirConfig.Path); err != nil {
						backends.Log().WithError(err).Errorf("554 Error: could not save email for [%s]", u)
						return backends.NewResult(response.Canned.FailBackendTransaction), err
					}
				}
				return c.Process(e, task)
			default:
				return c.Process(e, task)
			}
		})
	}
}
