### HOW-TO USE:
Post JSON to /scan:
```json
[
	{
		"name": "scanme.nmap.org",
		"ports": [22,80]
	},
	{
		"name": "www.ya.ru",
		"ports": [80,443]
	},
	{
		"name": "www.google.com",
		"ports": [80,443]
	},
	{
		"name": "getinside.cloud",
		"ports": [22,80,4443,8089,8390]
	}
]
```

```bash
curl --request POST --url http://localhost:8081/scan --header 'content-type: application/json' --data '[{"name": "scanme.nmap.org","ports": [22,80]},{"name": "www.ya.ru","ports": [80,443]},{"name": "www.google.com","ports": [80,443]},{"name": "getinside.cloud","ports": [22,80,4443,8089,8390]}]'
```