import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface SettingsState {
  theme: 'light' | 'dark';
  apiUrl: string;
}

const initialState: SettingsState = {
  theme: 'light',
  apiUrl: 'http://localhost:8080'
};

const settingsSlice = createSlice({
  name: 'settings',
  initialState,
  reducers: {
    setTheme(state, action: PayloadAction<'light' | 'dark'>) {
      state.theme = action.payload;
    },
    setApiUrl(state, action: PayloadAction<string>) {
      state.apiUrl = action.payload;
    }
  }
});

export const { setTheme, setApiUrl } = settingsSlice.actions;

export default settingsSlice.reducer;