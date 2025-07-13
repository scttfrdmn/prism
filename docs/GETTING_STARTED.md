# Getting Started with CloudWorkstation

Welcome to CloudWorkstation! This guide will help you create your very own powerful research computer in the cloud.

## What You'll Need

- A computer with internet access
- CloudWorkstation installed (ask your teacher or parent for help with this)
- An AWS account (this is what lets you create cloud computers)

## Step 1: Check That Everything Works

First, let's make sure CloudWorkstation is installed correctly:

```bash
cws version
```

You should see something like "CloudWorkstation v0.4.0" appear.

Now let's check if CloudWorkstation can talk to AWS:

```bash
cws test
```

If everything is working, you'll see a happy success message!

## Step 2: See What Research Environments Are Available

CloudWorkstation comes with pre-made research environments - like having different science labs ready to go!

```bash
cws templates
```

You'll see a list of available research environments like:
- **python-research**: For coding in Python and data science
- **r-research**: For statistics and data analysis
- **neuroimaging**: For brain research
- **bioinformatics**: For DNA and genetics research
- And more!

## Step 3: Launch Your Research Environment

Let's create a cloud computer with the Python research environment:

```bash
cws launch python-research my-first-project
```

CloudWorkstation will:
1. Find the best computer size for Python research
2. Start a new cloud computer
3. Set up all the Python tools automatically
4. Show you how to connect when it's ready

This might take about a minute - you'll see progress updates as it works.

## Step 4: Connect to Your Cloud Computer

Once your cloud computer is ready, you can connect to it:

```bash
cws connect my-first-project
```

CloudWorkstation will automatically open a connection for you! You'll see:

- The command line of your cloud computer, OR
- A web page with Jupyter Notebook or RStudio (depending on your environment)

## Step 5: Using Your Cloud Computer

Now you can use your cloud computer just like a normal computer, but with super powers!

- All the science tools you need are already installed
- You can upload your data files
- You can run complex calculations faster than on your regular computer
- You can save your work for next time

## Step 6: Turn Off Your Cloud Computer When You're Done

Cloud computers cost money when they're running, so it's important to turn them off when you're done:

```bash
cws stop my-first-project
```

Don't worry - your work will be saved and waiting for you next time!

## Step 7: Turn Your Cloud Computer Back On Later

When you want to work on your project again, just start it up:

```bash
cws start my-first-project
```

Then connect to it just like before:

```bash
cws connect my-first-project
```

## Cool Tip: Try the New TUI Interface!

CloudWorkstation has a colorful screen-based interface you can use instead of typing commands:

```bash
cws tui
```

You can use arrow keys to move around and select options. Press "?" anytime to see keyboard shortcuts!

## What If Something Goes Wrong?

If you have any problems, try these steps:

1. Check if CloudWorkstation is working:
   ```bash
   cws test
   ```

2. See if your cloud computer is running:
   ```bash
   cws list
   ```

3. Ask for help (your teacher, parent, or researcher friend)

## Next Steps

Once you're comfortable with the basics, you can:

- Try different research environments
- Create larger cloud computers for bigger projects
- Learn how to share your work with others
- Explore advanced features in the user guide

Happy researching! 🔬🚀