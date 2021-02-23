@echo off
copy ..\forcepser_config_exe\Win32_Release\forcepser_config.exe .\bin\
copy ..\forcepser_config_exe\Win32_Release\WebView2Loader.dll .\bin\
cd bin
..\7za.exe a -tzip ..\forcepser.zip forcepser.exe forcepser_config.exe WebView2Loader.dll forcepser.txt _entrypoint.lua setting.txt-template setting.txt-template-old asas
cd ..
