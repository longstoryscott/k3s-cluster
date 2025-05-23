import { FormControl, Grid, InputLabel, MenuItem, Select, SelectChangeEvent } from "@mui/material";
import { useEffect, useState } from "react";
import { useConfigContext } from "../../context/ConfigContext";
import { ModelProfile, ModelProfilesConfig } from "../../hooks/useConfig";
import { useAuth } from "../../auth";
import { listModelProfiles } from "../../api";

const ModelProfileSelector = ({ task }: { task: { key: keyof ModelProfilesConfig; label: string; }}) => {
  const { config, updateConfig } = useConfigContext();
  const [profiles, setProfiles] = useState<ModelProfile[]>([]);
  const auth = useAuth();
  const [value, setValue] = useState(config?.modelProfiles?.[task.key] || '');
  
  // Use effect to update value when config changes
  useEffect(() => {
    if (config?.modelProfiles && task.key in config.modelProfiles) {
      setValue(config.modelProfiles[task.key] || '');
    }
  }, [config, task.key]);

  // Fetch profiles on mount
  useEffect(() => {
    const fetchProfiles = async () => {
      try {
        // You may need to pass the token here
        const data = await listModelProfiles(auth.user.accessToken || '');
        setProfiles(data);
      } catch (err: unknown) {
        if (err instanceof Error) {
          console.error('Error fetching model profiles:', err.message);
        }
      }
    };
    fetchProfiles();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  
  const handleChange = (event: SelectChangeEvent) => {
    const newValue = event.target.value as string;
    setValue(newValue);
    if (config?.modelProfiles) {
      updateConfig({
        ...config,
        modelProfiles: {
          ...config.modelProfiles,
          [task.key]: newValue
        }
      });
    }
  };
  return (
    <Grid key={task.key} size={{ xs: 12, sm: 6, md: 4 }}>
      <FormControl fullWidth>
        <InputLabel>{task.label}</InputLabel>
        <Select
          value={value}
          onChange={handleChange}
          labelId={`${task.key}-select-label`}
          id={`${task.key}-select`}
          label={task.label}
        >
          <MenuItem value="">(None)</MenuItem>
          {profiles && profiles.map(profile => (
            <MenuItem key={profile.id} value={profile.id}>{profile.name}</MenuItem>
          ))}
        </Select>
      </FormControl>
    </Grid>
  )
}

export default ModelProfileSelector;