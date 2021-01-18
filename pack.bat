@echo off
cd bin
..\7za.exe a -tzip ..\forcepser.zip forcepser.exe forcepser.txt _entrypoint.lua setting.txt-template setting.txt-template-old asas
cd ..
