# CloudWorkstation School Pilot Quick Start Guide

*Last Updated: October 2025 • Version 0.5.5*

## 🎯 For Educational Institutions & School Pilots

This guide is specifically designed for educational institutions evaluating CloudWorkstation for their computing curriculum, research programs, and student projects. CloudWorkstation enables schools to provide students with professional-grade development environments without the complexity of traditional IT infrastructure.

## 📚 What is CloudWorkstation?

CloudWorkstation is an academic research platform that launches pre-configured cloud environments in seconds. Students and educators can access professional development tools, research environments, and collaborative workspaces through a simple interface - no IT expertise required.

**Perfect for:**
- Computer science courses and labs
- Data science and research projects
- Student coding assignments and portfolios
- Faculty research collaboration
- Cross-curricular technology integration

## ⚡ 5-Minute School Setup

### Step 1: Install CloudWorkstation

**For IT Administrators (Recommended)**
```bash
# Install via Homebrew (macOS/Linux)
brew tap scttfrdmn/cloudworkstation
brew install cloudworkstation

# Or download directly
curl -L https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation-darwin-arm64.tar.gz
```

**For Individual Educators**
- Download from [GitHub Releases](https://github.com/scttfrdmn/cloudworkstation/releases)
- Choose your platform: macOS, Linux, or Windows
- Extract and run - no complex installation required

### Step 2: AWS Account Setup (One-time per School)

CloudWorkstation uses AWS for cloud resources. Most schools can use AWS Educate for credits:

1. **Get AWS Account**:
   - Apply for [AWS Educate](https://aws.amazon.com/education/awseducate/) (free credits for schools)
   - Or use existing institutional AWS account
   - Individual educators can use personal AWS accounts for pilot testing

2. **Configure Credentials** (IT Admin):
```bash
# Install AWS CLI
brew install awscli  # or: pip install awscli

# Configure institutional credentials
aws configure
# AWS Access Key ID: [Your school's access key]
# AWS Secret Access Key: [Your school's secret key]
# Default region: us-west-2 (or closest to your location)
# Output format: json
```

### Step 3: Launch Your First Environment

**For Students/Educators** (Web Interface):
```bash
# Start the GUI application
cws-gui
```

**For Command Line Users**:
```bash
# View available templates
cws templates

# Launch a Python environment for data science
cws launch "Python Machine Learning" my-first-project

# Launch an R environment for statistics
cws launch "R Research Environment" statistics-project

# Launch basic Ubuntu for general computing
cws launch "Basic Ubuntu (APT)" cs-assignment
```

## 🎓 Educational Templates

CloudWorkstation includes pre-configured environments designed for educational use:

### **Python Machine Learning**
- **Best for**: Data science courses, AI/ML projects, research
- **Includes**: TensorFlow, PyTorch, Jupyter notebooks, pandas, scikit-learn
- **Launch time**: ~2 minutes
- **Cost**: ~$0.48/hour (AWS t3.medium)

### **R Research Environment**
- **Best for**: Statistics courses, data analysis, research projects
- **Includes**: RStudio, tidyverse, statistical packages
- **Launch time**: ~3 minutes
- **Cost**: ~$0.24/hour (AWS t3.small)

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
cws hibernate my-project  # Pause when not in use
cws resume my-project     # Resume with all work intact

# Use spot instances for assignments
cws launch "Python Machine Learning" assignment --spot

# Set up automatic hibernation policies
cws idle profile create classroom --idle-minutes 30 --action hibernate
```

## 👥 Classroom Management

### Multi-Student Support
```bash
# Launch environments for entire class
cws launch "Python Machine Learning" alice-ml-project
cws launch "Python Machine Learning" bob-ml-project
cws launch "Python Machine Learning" carol-ml-project

# Share files between students using EFS volumes
cws volume create class-shared-data
cws volume mount class-shared-data alice-ml-project
cws volume mount class-shared-data bob-ml-project
```

### Student Collaboration
- **Shared Storage**: Students can collaborate on projects through shared EFS volumes
- **Template Consistency**: All students use identical, professional environments
- **Easy Access**: Students connect via SSH or web-based tools (Jupyter, RStudio)

## 🔧 Common Educational Workflows

### **Computer Science Course**
```bash
# Launch basic Ubuntu environment for each student
cws launch "Basic Ubuntu (APT)" student-cs101

# Students get full Linux environment with:
# - GCC compiler, Python, Node.js, Git
# - File system access and admin privileges
# - Pre-configured development tools
# - SSH access for remote development
```

### **Data Science Class**
```bash
# Launch Python ML environment with Jupyter
cws launch "Python Machine Learning" student-datascience

# Students access via web browser:
# - Jupyter notebooks at http://[instance-ip]:8888
# - Pre-installed ML libraries and datasets
# - GPU acceleration available for advanced projects
# - Collaborative notebooks through shared storage
```

### **Research Project**
```bash
# Create shared research environment
cws volume create research-project-data
cws launch "R Research Environment" professor-research
cws launch "Python Machine Learning" student-researcher

# Mount shared storage for collaboration
cws volume mount research-project-data professor-research
cws volume mount research-project-data student-researcher

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
```bash
# Set up class-wide hibernation policies
cws idle profile create night-shutdown --idle-minutes 60 --action hibernate

# Apply to all student environments
cws idle bulk-apply night-shutdown cs101-*

# Automatic cost optimization without losing student work
```

### **Integration with LMS**
- **API Access**: Integrate with Canvas, Blackboard, Moodle via REST API
- **Single Sign-On**: Connect with school authentication systems (future)
- **Grade Integration**: Link projects with grading systems (future)

## 📋 Pilot Program Checklist

### Week 1: Setup & Testing
- [ ] Install CloudWorkstation on faculty machine
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
cws list
cws status my-project

# Restart if needed
cws start my-project
```

**"High costs"**
```bash
# Check running instances
cws list

# Hibernate unused environments
cws hibernate unused-project

# Set up automatic hibernation
cws idle profile create cost-saver --idle-minutes 15 --action hibernate
```

**"Student cannot access Jupyter/RStudio"**
```bash
# Get connection info
cws connect my-ml-project
# Follow the provided URL and SSH instructions
```

### Getting Help
- **Documentation**: [Full documentation](https://github.com/scttfrdmn/cloudworkstation/docs)
- **GitHub Issues**: [Report problems](https://github.com/scttfrdmn/cloudworkstation/issues)
- **Educational Support**: Email support@cloudworkstation.dev with "SCHOOL PILOT" in subject
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
2. **Faculty Training**: Train more educators on CloudWorkstation management
3. **Student Onboarding**: Create student-facing documentation and tutorials
4. **Integration Planning**: Connect with school's LMS and authentication systems

### Institutional Adoption
1. **IT Policy Integration**: Align with school's cloud and security policies
2. **Budget Planning**: Include CloudWorkstation in annual IT budget planning
3. **Curriculum Integration**: Update course syllabi to leverage cloud environments
4. **Research Enhancement**: Expand to faculty research projects and collaborations

---

## 📞 Pilot Program Support

**Ready to start your pilot?** Contact our education team:

- **Email**: education@cloudworkstation.dev
- **Subject**: School Pilot Program - [Your Institution]
- **Include**: School name, course details, expected student count, timeline

We provide:
- ✅ Free 30-day pilot support
- ✅ AWS credit guidance and optimization
- ✅ Custom template development for your curriculum
- ✅ Faculty training and documentation
- ✅ Student onboarding materials
- ✅ Success metrics and reporting

**Transform your computing education with CloudWorkstation - professional development environments for every student, managed with simplicity.**