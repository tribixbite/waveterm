# Building Wave Terminal on Windows

This document provides instructions for setting up the development environment for Wave Terminal on Windows.

## Prerequisites

Before you begin, ensure you have the following installed:
- Git
- Node.js (LTS version recommended)
- Yarn
- Go (version 1.18 or higher)

## Cloning the Repository

To clone the Wave Terminal repository, open a command prompt or PowerShell and run:

```
git clone https://github.com/tribixbite/waveterm.git
cd waveterm
```

## Setting Up the Development Environment

1. Install the required Node.js dependencies by running:

```
yarn install
```

2. Build the Electron application:

```
yarn run build:windows
```

This command will compile the TypeScript code and package the application for Windows.

3. To start the application in development mode, use:

```
yarn run start:windows
```

## Environment Variables and Path Adjustments

Ensure that the paths to Git, Node.js, Yarn, and Go are correctly set in your system's environment variables. This will allow you to run the necessary commands from any command prompt or PowerShell window.

## Troubleshooting

If you encounter issues during the setup, consider the following tips:

- Ensure all prerequisites are correctly installed and up to date.
- Check the environment variables to ensure paths are correctly set.
- For issues related to Yarn or Node.js dependencies, try removing the `node_modules` folder and running `yarn install` again.
- Consult the official documentation for Git, Node.js, Yarn, and Go for platform-specific setup instructions.

For further assistance, please open an issue on the GitHub repository.
