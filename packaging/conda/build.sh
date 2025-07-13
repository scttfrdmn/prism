#!/bin/bash

# Copy binaries to the conda environment bin directory
if [[ "$OSTYPE" == "darwin"* ]]; then
  # macOS
  cp cws ${PREFIX}/bin/
  cp cwsd ${PREFIX}/bin/
else
  # Linux
  cp cws ${PREFIX}/bin/
  cp cwsd ${PREFIX}/bin/
fi

# Make sure binaries are executable
chmod +x ${PREFIX}/bin/cws
chmod +x ${PREFIX}/bin/cwsd

# Generate shell completions
mkdir -p ${PREFIX}/etc/bash_completion.d/
${PREFIX}/bin/cws completion bash > ${PREFIX}/etc/bash_completion.d/cws

if [ -d ${PREFIX}/share/zsh/site-functions ]; then
  mkdir -p ${PREFIX}/share/zsh/site-functions/
  ${PREFIX}/bin/cws completion zsh > ${PREFIX}/share/zsh/site-functions/_cws
fi