@echo off
chcp 65001 >nul
cd /d "%~dp0"
title WeChat Obsidian Bot 安装向导
echo ========================================
echo   WeChat -^> Obsidian Bot 安装向导
echo ========================================
echo.
echo 这个脚本会帮你：检查环境、安装依赖、配置、激活
echo.
pause
wechat-obsidian-bot.exe --setup
