# Web Services Integration Guide

## Overview

This guide demonstrates how to integrate third-party web services and custom research tools into Prism using the template system. While Prism includes built-in support for AWS services like SageMaker, any web-accessible research tool can be integrated through templates.

## Integration Patterns

### **Pattern 1: Direct Web Service Templates**

For services that provide direct web access URLs:

```yaml
# Template: templates/custom-jupyter-hub.yml
name: "Custom JupyterHub Research Environment"
description: "Self-hosted JupyterHub with custom research libraries"
connection_type: "web"
service_type: "custom_web"

# Uses EC2 instance but configured for web access
base: "ubuntu-22.04"
packages:
  system: ["docker", "docker-compose", "nginx"]
  pip: ["jupyterhub", "dockerspawner", "oauthenticator"]

services:
  - name: "jupyterhub"
    port: 8000
    config:
      - "c.JupyterHub.hub_ip = '0.0.0.0'"
      - "c.JupyterHub.port = 8000"
      - "c.JupyterHub.spawner_class = 'dockerspawner.DockerSpawner'"

# Web service configuration
web_service:
  primary_port: 8000
  health_check_path: "/hub/health"
  access_path: "/hub"
  authentication_required: true

# Post-install configuration
post_install: |
  # Configure SSL certificate
  sudo certbot --nginx -d ${INSTANCE_DOMAIN}
  
  # Start JupyterHub service
  sudo systemctl enable jupyterhub
  sudo systemctl start jupyterhub

instance_defaults:
  type: "t3.large"  # Enough resources for multi-user access
  ports: [22, 80, 443, 8000]
```

### **Pattern 2: Containerized Research Tools**

For research tools distributed as Docker containers:

```yaml
# Template: templates/rstudio-server-custom.yml  
name: "RStudio Server with Custom Packages"
description: "Containerized RStudio Server with research-specific R packages"
connection_type: "web"
service_type: "docker_web"

base: "ubuntu-22.04"
packages:
  system: ["docker", "docker-compose"]

# Docker service definition
docker_services:
  rstudio:
    image: "rocker/rstudio:latest"
    ports:
      - "8787:8787"
    environment:
      - "DISABLE_AUTH=true"  # For research environment
      - "ROOT=TRUE"         # Allow root access
    volumes:
      - "/home/ubuntu/research:/home/rstudio/research"
      - "/mnt/shared-volume:/home/rstudio/shared"  # EFS integration
    
  # Custom R package installation container  
  r-packages:
    image: "r-base:latest"
    volumes:
      - "/home/ubuntu/r-libs:/usr/local/lib/R/site-library"
    command: |
      Rscript -e "
      install.packages(c('tidyverse', 'ggplot2', 'dplyr', 'shiny', 'plotly'))
      install.packages('BiocManager')
      BiocManager::install(c('DESeq2', 'edgeR', 'limma'))
      "

web_service:
  primary_port: 8787
  health_check_path: "/auth-sign-in"
  access_path: "/"
  
post_install: |
  # Wait for R package installation
  docker-compose up r-packages
  
  # Start RStudio Server
  docker-compose up -d rstudio
  
  # Configure reverse proxy for HTTPS
  sudo apt-get install -y nginx
  sudo systemctl enable nginx
```

### **Pattern 3: API-Driven Research Services**

For services that provide API endpoints:

```yaml
# Template: templates/mlflow-tracking-server.yml
name: "MLflow Tracking Server"
description: "Machine learning experiment tracking and model registry"  
connection_type: "web"
service_type: "api_web"

base: "ubuntu-22.04"
packages:
  system: ["python3", "python3-pip", "postgresql", "nginx"]
  pip: ["mlflow", "psycopg2-binary", "boto3"]

services:
  - name: "postgresql"
    port: 5432
    config:
      - "CREATE DATABASE mlflow;"
      - "CREATE USER mlflow WITH PASSWORD 'secure_password';"
      
  - name: "mlflow-server"
    port: 5000
    config: []

web_service:
  primary_port: 5000
  health_check_path: "/health"
  api_endpoints:
    - path: "/api/2.0/mlflow/experiments/list"
      method: "GET"
    - path: "/api/2.0/mlflow/runs/create"  
      method: "POST"
      
post_install: |
  # Configure MLflow with PostgreSQL backend
  export MLFLOW_BACKEND_STORE_URI="postgresql://mlflow:secure_password@localhost/mlflow"
  export MLFLOW_DEFAULT_ARTIFACT_ROOT="s3://mlflow-artifacts-${AWS_ACCOUNT_ID}"
  
  # Start MLflow server
  mlflow server \
    --backend-store-uri $MLFLOW_BACKEND_STORE_URI \
    --default-artifact-root $MLFLOW_DEFAULT_ARTIFACT_ROOT \
    --host 0.0.0.0 \
    --port 5000 &
    
  # Configure nginx reverse proxy
  sudo systemctl start nginx
```

## Integration with Prism Features

### **EFS Sharing Integration**

Web services can access shared EFS volumes:

```yaml
# In any web service template
post_install: |
  # Mount shared EFS volume for web service access
  if [ -n "$EFS_VOLUME_ID" ]; then
    sudo mkdir -p /var/web-service/shared
    sudo mount -t efs $EFS_VOLUME_ID:/ /var/web-service/shared
    
    # Configure web service to use shared storage
    echo "SHARED_DATA_PATH=/var/web-service/shared" >> /etc/web-service/config
  fi
```

### **Research User Integration**

Web services can authenticate with research users:

```yaml
post_install: |
  # Configure web service with research user authentication
  if [ -n "$RESEARCH_USER" ]; then
    # Add research user to web service group
    sudo usermod -a -G web-service-users $RESEARCH_USER
    
    # Configure web service authentication
    echo "AUTH_USER=$RESEARCH_USER" >> /etc/web-service/config
    echo "AUTH_HOME=/home/$RESEARCH_USER" >> /etc/web-service/config
  fi
```

### **Policy Framework Integration**

Web services inherit policy restrictions:

```yaml
# Policy restrictions apply to web services
policy_metadata:
  data_classification: "internal"
  suitable_for: ["research", "development"]
  resource_requirements:
    min_memory_gb: 8
    recommended_instance_types: ["t3.large", "m5.large"]
    
instance_defaults:
  type: "t3.large" 
  estimated_cost_per_hour:
    t3.large: 0.0832
```

## CLI Integration Examples

### **Launch and Access Web Services**

```bash
# Launch web service template
prism launch custom-jupyter-hub research-hub

# Check web service status
prism info research-hub
# Instance: research-hub  
# Service: Custom JupyterHub
# Status: Running
# Web URL: https://research-hub.cws.university.edu:8000/hub
# Health: ✓ Healthy (last checked: 30s ago)

# Open web service in browser
prism connect research-hub
# → Opens https://research-hub.cws.university.edu:8000/hub

# Get web service logs
prism logs research-hub --service jupyterhub
```

### **Custom Domain Integration**

```bash
# Configure custom domain for institution
prism domains add university.edu --verify-ownership
prism domains configure research-hub --domain research-tools.university.edu

# Launch with custom domain
prism launch rstudio-server-custom stats-analysis --domain stats.university.edu
# → Accessible at https://stats.university.edu
```

## Third-Party Service Examples

### **Computational Biology Tools**

```yaml
# Galaxy bioinformatics platform
name: "Galaxy Bioinformatics Workbench"
connection_type: "web"
service_type: "custom_web"

docker_services:
  galaxy:
    image: "quay.io/bgruening/galaxy-stable:latest"
    ports: ["8080:80"]
    volumes: 
      - "/mnt/galaxy-data:/export"
      - "/mnt/shared-volume:/import"  # EFS integration
```

### **Data Visualization Platforms**  

```yaml
# Observable/D3.js notebook server
name: "Observable Notebook Server"
connection_type: "web"
service_type: "custom_web"

packages:
  system: ["nodejs", "npm"]
  npm: ["@observablehq/runtime", "@observablehq/inspector"]

web_service:
  primary_port: 3000
  health_check_path: "/"
```

### **Collaborative Development**

```yaml
# Code-server (VS Code in browser)
name: "VS Code Server Research Environment"
connection_type: "web"
service_type: "custom_web"

packages:
  system: ["curl", "wget"]
  
post_install: |
  # Install code-server
  curl -fsSL https://code-server.dev/install.sh | sh
  
  # Configure with research user
  sudo systemctl enable --now code-server@$RESEARCH_USER
  
  # Install research extensions
  code-server --install-extension ms-python.python
  code-server --install-extension ms-toolsai.jupyter
```

## Best Practices

### **Security Considerations**

1. **Authentication**: Always implement proper authentication for web services
2. **SSL/TLS**: Use HTTPS for all web service access  
3. **Firewall**: Restrict ports to necessary services only
4. **Updates**: Keep web service containers/packages updated

### **Performance Optimization**

1. **Resource Sizing**: Choose appropriate instance types for web service workloads
2. **Caching**: Implement caching for frequently accessed data
3. **Load Balancing**: Use nginx for reverse proxy and load balancing
4. **Auto-scaling**: Consider auto-scaling for high-demand services

### **Integration Patterns**

1. **EFS Integration**: Always mount shared EFS for collaborative access
2. **Research User**: Configure services to work with research user identity  
3. **Policy Compliance**: Respect Prism policy restrictions
4. **Health Checks**: Implement proper health check endpoints

This guide enables researchers and institutions to integrate any web-based research tool into Prism while maintaining the platform's governance, security, and user experience benefits.