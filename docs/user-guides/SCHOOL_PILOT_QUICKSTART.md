# Prism School Pilot Quick Start Guide

*Last Updated: March 2026*

## 🎯 For Educational Institutions & School Pilots

This guide is specifically designed for educational institutions evaluating Prism for their computing curriculum, research programs, and student projects. Prism enables schools to provide students with professional-grade development environments without the complexity of traditional IT infrastructure.

## 📚 What is Prism?

Prism is an academic research platform that provides pre-configured cloud environments for students and educators. Access professional development tools, research environments, and collaborative workspaces through a simple interface - no IT expertise required.

**Perfect for:**
- Computer science courses and labs
- Data science and research projects
- Student coding assignments and portfolios
- Faculty research collaboration
- Cross-curricular technology integration

## ⚡ 5-Minute School Setup

### Step 1: Install Prism

```bash
# macOS
brew install scttfrdmn/tap/prism

# Windows
scoop bucket add scttfrdmn https://github.com/scttfrdmn/scoop-bucket
scoop install prism
```

See the [Installation guide](INSTALLATION.md) for all platforms and the desktop
app.

### Step 2: AWS Account Setup (One-time per School)

Prism uses AWS for cloud resources. Most schools can use AWS Educate for credits:

1. **Get AWS Account**:
   - Apply for [AWS Educate](https://aws.amazon.com/education/awseducate/) (free credits for schools)
   - Or use existing institutional AWS account
   - Individual educators can use personal AWS accounts for pilot testing

2. **Configure Credentials** (IT Admin):
```bash
# Install AWS CLI
brew install awscli  # or: pip install awscli

# Configure institutional credentials
aws login
# AWS Access Key ID: [Your school's access key]
# AWS Secret Access Key: [Your school's secret key]
# Default region: us-west-2 (or closest to your location)
# Output format: json
```

### Step 3: Launch Your First Environment

**For Students/Educators** (Web Interface):
```bash
# Start the GUI application
prism gui
```

**For Command Line Users**:
```bash
# View available templates
prism templates

# Launch a Python environment for data science
prism workspace launch python-ml my-first-project

# Launch an R environment for statistics
prism workspace launch r-rstudio-server statistics-project

# Launch basic Ubuntu for general computing
prism workspace launch basic-ubuntu cs-assignment
```

## 🎓 Educational Templates

Prism includes pre-configured environments designed for educational use:

### **Python Machine Learning** (`python-ml`)
- **Best for**: Data science courses, AI/ML projects, research
- **Includes**: TensorFlow, PyTorch, Jupyter notebooks, pandas, scikit-learn
- **Launch time**: ~5 minutes
- **Cost**: ~$0.042/hour (AWS t3.medium)

### **R + RStudio Server** (`r-rstudio-server`)
- **Best for**: Statistics courses, data analysis, research projects
- **Includes**: RStudio, tidyverse, R 4.4+, statistical packages
- **Launch time**: ~8 minutes
- **Cost**: ~$0.042/hour (AWS t3.medium)

### **Basic Ubuntu (APT)**
- **Best for**: Computer science fundamentals, programming courses
- **Includes**: Ubuntu Linux, development tools, package management
- **Launch time**: ~1 minute
- **Cost**: ~$0.12/hour (AWS t3.micro)

### **Web Development**
- **Best for**: Web design courses, full-stack development
- **Includes**: Node.js, Python, development tools, web servers
- **Launch time**: ~2 minutes
- **Cost**: ~$0.36/hour (AWS t3.small)

## 💰 Cost Management for Schools

### Budget-Friendly Features
- **Automatic Hibernation**: Environments automatically pause when idle (preserves student work, minimal cost)
- **Spot Instances**: Use spare AWS capacity for 60-90% cost savings
- **Right-sizing**: Templates automatically choose cost-effective instance types
- **Usage Tracking**: Monitor spending across classes and projects

### Example Monthly Costs (30 hours of use per student):
- **Basic Ubuntu**: ~$3.60/student/month
- **Python ML Environment**: ~$14.40/student/month
- **R Research Environment**: ~$7.20/student/month

### Cost Optimization Tips:
```bash
# Enable hibernation for classes (preserves student work)
prism workspace hibernate my-project  # Pause when not in use
prism workspace resume my-project     # Resume with all work intact

# Use spot instances for assignments
prism workspace launch python-ml assignment --spot

# Set up automatic hibernation policies
prism idle profile create classroom --idle-minutes 30 --action hibernate
```

## 👥 Classroom Management

### Multi-Student Support
```bash
# Launch environments for entire class
prism workspace launch python-ml alice-ml-project
prism workspace launch python-ml bob-ml-project
prism workspace launch python-ml carol-ml-project

# Share files between students using EFS volumes
prism volume create class-shared-data
prism volume attach class-shared-data alice-ml-project
prism volume attach class-shared-data bob-ml-project
```

### Student Collaboration
- **Shared Storage**: Students can collaborate on projects through shared EFS volumes
- **Template Consistency**: All students use identical, professional environments
- **Easy Access**: Students connect via SSH or web-based tools (Jupyter, RStudio)

## 🔧 Common Educational Workflows

### **Computer Science Course**
```bash
# Launch basic Ubuntu environment for each student
prism workspace launch basic-ubuntu student-cs101

# Students get full Linux environment with:
# - GCC compiler, Python, Node.js, Git
# - File system access and admin privileges
# - Pre-configured development tools
# - SSH access for remote development
```

### **Data Science Class**
```bash
# Launch Python ML environment with Jupyter
prism workspace launch python-ml student-datascience

# Students access via web browser:
# - Jupyter notebooks at http://[instance-ip]:8888
# - Pre-installed ML libraries and datasets
# - GPU acceleration available for advanced projects
# - Collaborative notebooks through shared storage
```

### **Research Project**
```bash
# Create shared research environment
prism volume create research-project-data
prism workspace launch r-rstudio-server professor-research
prism workspace launch python-ml student-researcher

# Attach shared storage for collaboration
prism volume attach research-project-data professor-research
prism volume attach research-project-data student-researcher

# Both can access shared data and collaborate in real-time
```

## 🔒 Security & Privacy for Schools

### Student Data Protection
- **Isolated Environments**: Each student gets private, isolated cloud environment
- **No Shared Infrastructure**: Students cannot access each other's work unless explicitly shared
- **Automatic Cleanup**: Environments can be automatically terminated at semester end
- **Backup Integration**: Student work automatically backed up to AWS EBS volumes

### FERPA Compliance
- **Private by Default**: Student environments are private and encrypted
- **Access Controls**: Only authorized faculty can manage student environments
- **Audit Logging**: Complete logs of all environment access and changes
- **Data Retention**: Configurable data retention policies for academic records

## 🚀 Advanced Features for Educators

### **Professional GUI Interface**
- **Visual Management**: Point-and-click interface for non-technical staff
- **Real-time Monitoring**: See all student environments and their status
- **Cost Dashboard**: Track usage and spending across classes
- **Template Marketplace**: Access community-contributed educational templates

### **Automated Management**

Templates include built-in idle detection that automatically hibernates instances after a configurable period of inactivity (default: 60 minutes). This preserves student work while reducing costs — no manual setup required.

To hibernate all student environments manually:
```bash
prism workspace hibernate student-name-project
```

### **Integration with LMS**
- **API Access**: Integrate with Canvas, Blackboard, Moodle via REST API
- **Single Sign-On**: Connect with school authentication systems (future)
- **Grade Integration**: Link projects with grading systems (future)

## 📋 Pilot Program Checklist

### Week 1: Setup & Testing
- [ ] Install Prism on faculty machine
- [ ] Configure AWS account with educational credits
- [ ] Test launch of all relevant templates
- [ ] Verify cost monitoring and hibernation
- [ ] Document any issues or questions

### Week 2: Small Class Pilot
- [ ] Launch environments for 3-5 students
- [ ] Test collaboration features (shared volumes)
- [ ] Monitor costs and usage patterns
- [ ] Gather student feedback on usability
- [ ] Document workflow improvements

### Week 3: Full Class Deployment
- [ ] Scale to full class size (20-30 students)
- [ ] Implement automated policies (hibernation, cost controls)
- [ ] Integration testing with existing curriculum
- [ ] Staff training for ongoing management
- [ ] Prepare expansion plan for other courses

### Week 4: Evaluation & Planning
- [ ] Cost analysis vs traditional lab infrastructure
- [ ] Student learning outcome assessment
- [ ] Faculty productivity improvements
- [ ] Technical support requirements
- [ ] Plan for full institutional adoption

## 🆘 Quick Troubleshooting

### Common Issues

**"Cannot connect to environment"**
```bash
# Check if environment is running
prism workspace list

# Resume if stopped
prism workspace resume my-project
```

**"High costs"**
```bash
# Check running instances
prism workspace list

# Hibernate unused environments
prism workspace hibernate unused-project
```

**"Student cannot access Jupyter/RStudio"**
```bash
# Get connection info
prism workspace connect my-ml-project
# Follow the provided URL and SSH instructions
```

### Getting Help
- **Documentation**: [Full documentation](https://github.com/scttfrdmn/prism/docs)
- **GitHub Issues**: [Report problems](https://github.com/scttfrdmn/prism/issues)
- **Educational Support**: Email support@prism.dev with "SCHOOL PILOT" in subject
- **Community Forum**: Join educator discussions (coming soon)

## 📈 Success Metrics

Track these metrics during your pilot:

### **Student Engagement**
- Time spent in development environments
- Number of projects completed
- Code commits and collaboration activity
- Student satisfaction surveys

### **Educational Outcomes**
- Project complexity and quality improvements
- Reduced setup time (from hours to minutes)
- Increased focus on learning vs troubleshooting
- Cross-platform consistency (all students same environment)

### **Operational Efficiency**
- IT support ticket reduction
- Faculty time saved on environment setup
- Cost savings vs traditional computer labs
- Scalability to other courses and departments

### **Cost Analysis**
- AWS costs per student per course
- Comparison to hardware lab costs
- ROI calculation including faculty time savings
- Hibernation effectiveness (cost reduction)

## 🌟 Next Steps After Pilot

### Successful Pilot Outcomes
1. **Expand to More Courses**: Roll out to additional CS, data science, and research courses
2. **Faculty Training**: Train more educators on Prism management
3. **Student Onboarding**: Create student-facing documentation and tutorials
4. **Integration Planning**: Connect with school's LMS and authentication systems

### Institutional Adoption
1. **IT Policy Integration**: Align with school's cloud and security policies
2. **Budget Planning**: Include Prism in annual IT budget planning
3. **Curriculum Integration**: Update course syllabi to leverage cloud environments
4. **Research Enhancement**: Expand to faculty research projects and collaborations

---

## 📞 Pilot Program Support

**Ready to start your pilot?** Contact our education team:

- **Email**: education@prism.dev
- **Subject**: School Pilot Program - [Your Institution]
- **Include**: School name, course details, expected student count, timeline

We provide:
- ✅ Free 30-day pilot support
- ✅ AWS credit guidance and optimization
- ✅ Custom template development for your curriculum
- ✅ Faculty training and documentation
- ✅ Student onboarding materials
- ✅ Success metrics and reporting

**Transform your computing education with Prism - professional development environments for every student, managed with simplicity.**