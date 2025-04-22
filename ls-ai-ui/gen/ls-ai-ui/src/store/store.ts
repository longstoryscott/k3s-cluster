import { configureStore } from '@reduxjs/toolkit';
import chatReducer from './chatSlice';
import settingsReducer from './settingsSlice';

const store = configureStore({
  reducer: {
    chat: chatReducer,
    settings: settingsReducer
  }
});

export default store;