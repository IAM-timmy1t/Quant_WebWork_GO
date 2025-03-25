#!/bin/bash

# Bridge Module Setup Script
# This script installs all required dependencies and sets up the development environment

# Color output for better readability
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Setting up Quant WebWorks GO Bridge Module...${NC}"

# Install npm dependencies
echo -e "${GREEN}Installing npm dependencies...${NC}"
npm install || { echo -e "${RED}Failed to install npm dependencies${NC}"; exit 1; }

# Restore missing dependencies (checking if we need to add missing protobuf dependencies that might have been removed)
echo -e "${GREEN}Checking for missing dependencies...${NC}"
if ! grep -q "@improbable-eng/grpc-web" package.json; then
  echo -e "${YELLOW}Restoring gRPC Web dependencies...${NC}"
  npm install --save @improbable-eng/grpc-web@0.15.0
fi

if ! grep -q "google-protobuf" package.json; then
  echo -e "${YELLOW}Restoring Protocol Buffers dependencies...${NC}"
  npm install --save google-protobuf@3.21.2
fi

if ! grep -q "events" package.json; then
  echo -e "${YELLOW}Restoring events dependency...${NC}"
  npm install --save events@3.3.0
fi

# Setup testing environment
echo -e "${GREEN}Setting up testing environment...${NC}"
# Create cypress directory if it doesn't exist
if [ ! -d "cypress" ]; then
  echo -e "${YELLOW}Creating Cypress directory structure...${NC}"
  mkdir -p cypress/e2e
  mkdir -p cypress/fixtures
  mkdir -p cypress/support
  
  # Create default support files
  echo "// ***********************************************************
// This support file is processed and loaded automatically 
// before your test files.
// ***********************************************************

// Import commands.js using ES2015 syntax:
import './commands'

// Alternatively you can use CommonJS syntax:
// require('./commands')" > cypress/support/e2e.js
  
  echo "// ***********************************************
// Custom commands for Cypress tests
// ***********************************************

// Example custom command:
// Cypress.Commands.add('login', (email, password) => { ... })" > cypress/support/commands.js
fi

# Create cypress.config.js if it doesn't exist
if [ ! -f "cypress.config.js" ]; then
  echo -e "${YELLOW}Creating Cypress configuration...${NC}"
  echo "const { defineConfig } = require('cypress')

module.exports = defineConfig({
  e2e: {
    setupNodeEvents(on, config) {
      // implement node event listeners here
    },
    baseUrl: 'http://localhost:3000',
    specPattern: 'cypress/e2e/**/*.{js,jsx,ts,tsx}',
    viewportWidth: 1280,
    viewportHeight: 720
  },
})" > cypress.config.js
fi

# Create tsconfig.json if it doesn't exist
if [ ! -f "tsconfig.json" ]; then
  echo -e "${YELLOW}Creating TypeScript configuration...${NC}"
  echo '{
  "compilerOptions": {
    "target": "es5",
    "lib": [
      "dom",
      "dom.iterable",
      "esnext"
    ],
    "allowJs": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "allowSyntheticDefaultImports": true,
    "strict": true,
    "forceConsistentCasingInFileNames": true,
    "noFallthroughCasesInSwitch": true,
    "module": "esnext",
    "moduleResolution": "node",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "baseUrl": "src",
    "paths": {
      "@/*": ["*"]
    },
    "types": ["jest", "node", "@testing-library/jest-dom"]
  },
  "include": [
    "src",
    "cypress"
  ]
}' > tsconfig.json
fi

# Create setupTests.js if it doesn't exist
if [ ! -f "src/setupTests.ts" ]; then
  echo -e "${YELLOW}Creating test setup file...${NC}"
  mkdir -p src
  echo "// jest-dom adds custom jest matchers for asserting on DOM nodes.
// allows you to do things like:
// expect(element).toHaveTextContent(/react/i)
// learn more: https://github.com/testing-library/jest-dom
import '@testing-library/jest-dom';" > src/setupTests.ts
fi

echo -e "${GREEN}Setup completed successfully!${NC}"
echo -e "${YELLOW}Run 'npm start' to start the development server${NC}"
echo -e "${YELLOW}Run 'npm test' to run unit tests${NC}"
echo -e "${YELLOW}Run 'npm run test:e2e' to run Cypress end-to-end tests${NC}"
