import './App.css'
import { Box, Button, Grid, Paper, TextField, Typography, useTheme } from "@mui/material";
import * as React from "react";
import { gen } from "./api";
import { useAuth } from './auth';
import ReactMarkdown from 'react-markdown';

function Prompt() {
  const [prompt, setPrompt] = React.useState('');
  const [response, setResponse] = React.useState('');
  const [_, setContext] = React.useState('');
  // @ts-expect-error @typescript-eslint/no-unused-vars
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [model, setModel] = React.useState('phi3.5');
  const auth = useAuth();
  const theme = useTheme();

  const generateResponse = async () => {
    const generator = gen({
      body: JSON.stringify({
        model: model,
        messages: [
          {
            role: 'user',
            content: prompt
          }
        ]
      }),
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${auth.user?.accessToken}`,
        'Content-Type': 'application/json'
      },
      path: 'api/chat'
    });
    
    for await (const res of generator) {
      setResponse((prev) => prev + res.message?.content);
      if (res.done) {
        setResponse((prev) => prev + res.message?.content);
        break;
      }
      if (res.context) {
        setContext(res.context.join(','));
      }
    }
  }
  const keyPressed = async (event: React.KeyboardEvent) => {
    if (event.key === 'Enter') {
      await generateResponse();
    }
  }

  return (
    <Grid container spacing={2} sx={{ padding: theme.spacing(2.5) }}>
      <TextField
        placeholder="Prompt"
        color="secondary"
        fullWidth
        size="medium"
        type="text"
        variant="filled"
        value={prompt}
        onChange={(event: React.ChangeEvent<HTMLInputElement>) => {
          setPrompt(event.target.value);
        }}
        onKeyDown={keyPressed}
      />
      <Button onClick={generateResponse} type='submit'>Generate</Button>

      {response && (
        <Paper elevation={2} sx={{ p: theme.spacing(2), mt: theme.spacing(2) }}>
          <Typography variant="h6" gutterBottom>Response:</Typography>
          <Box className="markdown-response">
            <ReactMarkdown>
              {response}
            </ReactMarkdown>
          </Box>
        </Paper>
      )}
    </Grid>
  )
}

export default Prompt
