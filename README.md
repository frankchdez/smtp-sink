# smtp-sink
Simple smtp server that strips attachments and stores them in a dir

#### Generate cert

```
$ sudo openssl req -newkey rsa:4096 -nodes -sha512 -x509 -days 3650 -nodes -out /etc/ssl/certs/mailserver.pem -keyout /etc/ssl/certs/mailserver.key
```
