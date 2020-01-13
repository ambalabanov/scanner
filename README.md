### Алгоритм работы:
1. Хосты для сканирования берутся из вывода NMAP (файл _nmap_output.xml_) или из файла _config.json_.
Файл _nmap_output.xml_ можно сформировать командой `nmap --open -p- -i nmap_input.txt -oX nmap_output.xml`
2. Проводятся попытки подключение к хостам по HTTP, в случае успеха результат записывается в mongodb

