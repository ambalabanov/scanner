## HOW-TO USE:
### Create scan

POST plaintext to /api/v1/parse:
```text
www.yandex.ru
scanme.nmap.org
```
or 

POST JSON to /api/v1/url:
```json
[
	{"url": "https://www.yandex.ru"},
	{"url": "http://scanme.nmap.org"}
]
```

### Get results
GET /api/v1/parse

### Delete results
DELETE /api/v1/parse