@echo off
cd bin
..\7za.exe a -tzip ..\forcepser.zip forcepser.exe forcepser.txt _entrypoint.lua template.exo-template setting.txt-template asas
cd ..
