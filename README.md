### HOW-TO USE:
1. Prepare _config.json_.
2. If you want use NMAP output, create _nmap.xml_ with command `nmap --open -p- scanme.nmap.org -oX nmap.xml` and set _use_ parameter in config.
3. Post JSON:
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