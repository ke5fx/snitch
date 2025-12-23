@echo off
cl /nologo /O2 /W3 /EHsc snitch.cpp /Fe:snitch.exe /link iphlpapi.lib ws2_32.lib
