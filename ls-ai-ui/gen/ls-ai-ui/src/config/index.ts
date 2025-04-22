import { darkTheme, lightTheme } from "./theme";

export default {
  server: {
    baseUrl: import.meta.env.VITE_BASE_URL
  },
  auth: {
    clientId: import.meta.env.VITE_CLIENT_ID,
    clientSecret: import.meta.env.VITE_CLIENT_SECRET,
    tokenEndpoint: `${import.meta.env.VITE_ISSUER}/token`,
    logoutEndpoint: `${import.meta.env.VITE_ISSUER}/logout`,
    scope: 'openid profile email offline_access',
    tokenStorageKey: 'auth_tokens',
    userStorageKey: 'auth_user'
  },
  theme: {
    light: lightTheme,
    dark: darkTheme
  }
}