@echo off
mkdir %PREFIX%\bin
copy cws.exe %PREFIX%\bin\
copy cwsd.exe %PREFIX%\bin\

if exist completions (
  mkdir %PREFIX%\etc\bash_completion.d
  copy completions\cws.bash %PREFIX%\etc\bash_completion.d\cws

  mkdir %PREFIX%\share\zsh\site-functions
  copy completions\cws.zsh %PREFIX%\share\zsh\site-functions\_cws

  mkdir %PREFIX%\share\fish\vendor_completions.d
  copy completions\cws.fish %PREFIX%\share\fish\vendor_completions.d\
)

if exist man (
  mkdir %PREFIX%\share\man\man1
  copy man\cws.1 %PREFIX%\share\man\man1\
  copy man\cwsd.1 %PREFIX%\share\man\man1\
)