#!/bin/bash

mkdir -p $PREFIX/bin
cp cws $PREFIX/bin/
cp cwsd $PREFIX/bin/

# Copy completion files if they exist
if [ -d completions ]; then
  mkdir -p $PREFIX/etc/bash_completion.d
  cp completions/cws.bash $PREFIX/etc/bash_completion.d/cws

  mkdir -p $PREFIX/share/zsh/site-functions
  cp completions/cws.zsh $PREFIX/share/zsh/site-functions/_cws

  mkdir -p $PREFIX/share/fish/vendor_completions.d
  cp completions/cws.fish $PREFIX/share/fish/vendor_completions.d/
fi

# Copy man pages if they exist
if [ -d man ]; then
  mkdir -p $PREFIX/share/man/man1
  cp man/cws.1 $PREFIX/share/man/man1/
  cp man/cwsd.1 $PREFIX/share/man/man1/
fi