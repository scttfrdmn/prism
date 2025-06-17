package aws

import "github.com/scttfrdmn/cloudworkstation/pkg/types"

// getTemplates returns the hard-coded templates for MVP
func getTemplates() map[string]types.Template {
	return map[string]types.Template{
		"r-research": {
			Name:        "R Research Environment",
			Description: "R + RStudio Server + tidyverse packages",
			AMI: map[string]map[string]string{
				"us-east-1": {
					"x86_64": "ami-02029c87fa31fb148",
					"arm64":  "ami-050499786ebf55a6a",
				},
				"us-east-2": {
					"x86_64": "ami-0b05d988257befbbe",
					"arm64":  "ami-010755a3881216bba",
				},
				"us-west-1": {
					"x86_64": "ami-043b59f1d11f8f189",
					"arm64":  "ami-0d3e8bea392f79ebb",
				},
				"us-west-2": {
					"x86_64": "ami-016d360a89daa11ba",
					"arm64":  "ami-09f6c9efbf93542be",
				},
			},
			InstanceType: map[string]string{
				"x86_64": "t3.medium",
				"arm64":  "t4g.medium", // Graviton2 ARM-based
			},
			UserData: `#!/bin/bash
apt update -y
apt install -y r-base r-base-dev

# Detect architecture and install appropriate RStudio Server
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    wget https://download2.rstudio.org/server/jammy/amd64/rstudio-server-2023.06.1-524-amd64.deb
    dpkg -i rstudio-server-2023.06.1-524-amd64.deb || true
elif [ "$ARCH" = "aarch64" ]; then
    wget https://download2.rstudio.org/server/jammy/arm64/rstudio-server-2023.06.1-524-arm64.deb
    dpkg -i rstudio-server-2023.06.1-524-arm64.deb || true
fi
apt-get install -f -y

# Install common R packages
R -e "install.packages(c('tidyverse','ggplot2','dplyr','readr'), repos='http://cran.rstudio.com/')"
# Configure RStudio
echo "www-port=8787" >> /etc/rstudio/rserver.conf
systemctl restart rstudio-server
# Create ubuntu user for RStudio
echo "ubuntu:password123" | chpasswd
echo "Setup complete" > /var/log/cws-setup.log
`,
			Ports: []int{22, 8787},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.0464,
				"arm64":  0.0368, // Graviton2 is typically 20% cheaper
			},
		},
		"python-research": {
			Name:        "Python Research Environment",
			Description: "Python + Jupyter + data science packages",
			AMI: map[string]map[string]string{
				"us-east-1": {
					"x86_64": "ami-02029c87fa31fb148",
					"arm64":  "ami-050499786ebf55a6a",
				},
				"us-east-2": {
					"x86_64": "ami-0b05d988257befbbe",
					"arm64":  "ami-010755a3881216bba",
				},
				"us-west-1": {
					"x86_64": "ami-043b59f1d11f8f189",
					"arm64":  "ami-0d3e8bea392f79ebb",
				},
				"us-west-2": {
					"x86_64": "ami-016d360a89daa11ba",
					"arm64":  "ami-09f6c9efbf93542be",
				},
			},
			InstanceType: map[string]string{
				"x86_64": "t3.medium",
				"arm64":  "t4g.medium",
			},
			UserData: `#!/bin/bash
apt update -y
apt install -y python3 python3-pip
pip3 install jupyter pandas numpy matplotlib seaborn scikit-learn
# Configure Jupyter
mkdir -p /home/ubuntu/.jupyter
cat > /home/ubuntu/.jupyter/jupyter_notebook_config.py << 'JUPYTER_EOF'
c.NotebookApp.ip = '0.0.0.0'
c.NotebookApp.port = 8888
c.NotebookApp.open_browser = False
c.NotebookApp.token = ''
c.NotebookApp.password = ''
JUPYTER_EOF
chown -R ubuntu:ubuntu /home/ubuntu/.jupyter
# Start Jupyter as service
sudo -u ubuntu nohup jupyter notebook --config=/home/ubuntu/.jupyter/jupyter_notebook_config.py > /var/log/jupyter.log 2>&1 &
echo "Setup complete" > /var/log/cws-setup.log
`,
			Ports: []int{22, 8888},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.0464,
				"arm64":  0.0368,
			},
		},
		"basic-ubuntu": {
			Name:        "Basic Ubuntu",
			Description: "Plain Ubuntu 22.04 for general use",
			AMI: map[string]map[string]string{
				"us-east-1": {
					"x86_64": "ami-02029c87fa31fb148",
					"arm64":  "ami-050499786ebf55a6a",
				},
				"us-east-2": {
					"x86_64": "ami-0b05d988257befbbe",
					"arm64":  "ami-010755a3881216bba",
				},
				"us-west-1": {
					"x86_64": "ami-043b59f1d11f8f189",
					"arm64":  "ami-0d3e8bea392f79ebb",
				},
				"us-west-2": {
					"x86_64": "ami-016d360a89daa11ba",
					"arm64":  "ami-09f6c9efbf93542be",
				},
			},
			InstanceType: map[string]string{
				"x86_64": "t3.micro",
				"arm64":  "t4g.micro",
			},
			UserData: `#!/bin/bash
apt update -y
echo "Setup complete" > /var/log/cws-setup.log
`,
			Ports: []int{22},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.0116,
				"arm64":  0.0092,
			},
		},
	}
}