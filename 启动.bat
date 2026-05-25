@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo ========================================
echo   WeChat -^> Obsidian 同步机器人
echo ========================================
echo.
wechat-obsidian-bot.exe
pause
