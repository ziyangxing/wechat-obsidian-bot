@echo off
chcp 65001 >nul
cd /d "%~dp0"
wechat-obsidian-bot.exe --device-code
pause
