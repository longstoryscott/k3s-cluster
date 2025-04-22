# ls-ai-ui

## Overview
This project is a web-based chat application that utilizes AI for generating responses. It is built using React and TypeScript, and it follows a modular architecture to facilitate maintainability and scalability.

## Features
- User authentication with JWT tokens.
- Real-time chat interface with typing indicators.
- Support for multiple AI models.
- Chat history management.
- Responsive design with light and dark themes.
- Settings for API configuration and user profiles.

## Project Structure
The project is organized into several directories, each serving a specific purpose:

- **public/**: Contains static assets like the favicon and the main HTML file.
- **src/**: The main source code for the application.
  - **api/**: Functions and types for making API calls.
  - **assets/**: Icon assets used throughout the application.
  - **auth/**: Authentication context and hooks.
  - **components/**: Reusable UI components.
    - **Chat/**: Components related to the chat interface.
    - **Layout/**: Components for the application layout.
    - **ModelSelector/**: Components for selecting AI models.
    - **Settings/**: Components for managing application settings.
    - **Shared/**: Shared components like buttons and loaders.
    - **Sidebar/**: Components for the sidebar navigation.
  - **config/**: Configuration settings for the application.
  - **constants/**: Constant values used throughout the application.
  - **context/**: Context for managing chat state.
  - **hooks/**: Custom hooks for various functionalities.
  - **pages/**: Main pages of the application.
  - **store/**: Redux store configuration and slices.
  - **styles/**: Global styles and theme settings.
  - **types/**: TypeScript types used throughout the application.
  - **utils/**: Utility functions for various operations.
  - **App.tsx**: Main application component.
  - **index.css**: Global CSS styles.
  - **main.tsx**: Entry point for the React application.

## Installation
1. Clone the repository:
   ```
   git clone <repository-url>
   ```
2. Navigate to the project directory:
   ```
   cd ls-ai-ui
   ```
3. Install dependencies:
   ```
   npm install
   ```
4. Create a `.env` file in the root directory and add the necessary environment variables:
   ```
   VITE_BASE_URL='http://localhost:8080'
   VITE_ISSUER='https://auth.longstorymedia.com'
   VITE_CLIENT_ID='public-client'
   ```
5. Start the development server:
   ```
   npm run dev
   ```

## Usage
- Navigate to the application in your web browser.
- Use the login form to authenticate.
- Once logged in, you can start chatting with the AI.

## Contributing
Contributions are welcome! Please open an issue or submit a pull request for any enhancements or bug fixes.

## License
This project is licensed under the MIT License. See the LICENSE file for more details.