import { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../auth';
import { getConfig, getToken, updateConfig } from '../api';

// Define TypeScript interfaces for our configuration structure matching the backend
interface SummarizationConfig {
  enabled?: boolean;
  messagesBeforeSummary?: number;
  summariesBeforeConsolidation?: number;
  embeddingDimension?: number;
  enableRAG?: boolean;
  enableResponseFiltering?: boolean;
  enableResponseCritique?: boolean;
  maxSummaryLevels?: number;
  summaryWeightCoefficient?: number;
}

interface RetrievalConfig {
  enabled?: boolean;
  limit?: number;
  enableCrossConversation?: boolean;
  similarityThreshold?: number;
  alwaysRetrieve?: boolean;
}

interface PreferencesConfig {
  defaultModel?: string;
  theme?: string;
  fontSize?: number;
  notificationsOn?: boolean;
  language?: string;
}

interface WebSearchConfig {
  enabled?: boolean;
  autoDetect?: boolean;
  maxResults?: number;
  includeResults?: boolean;
}

export interface ModelProfile {
  id: string;
  userId: string;
  name: string;
  description?: string;
  modelName: string;
  parameters: {
    num_ctx?: number;
    repeat_last_n?: number;
    repeat_penalty?: number;
    temperature?: number;
    seed?: number;
    stop?: string;
    num_predict?: number;
    top_k?: number;
    top_p?: number;
    min_p?: number;
  };
  systemPrompt: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface ModelProfilesConfig {
  primaryProfileId?: string;
  summarizationProfileId?: string;
  masterSummaryProfileId?: string;
  briefSummaryProfileId?: string;
  keyPointsProfileId?: string;
  selfCritiqueProfileId?: string;
  improvementProfileId?: string;
  memoryRetrievalProfileId?: string;
  analysisProfileId?: string;
  researchTaskProfileId?: string;
  researchPlanProfileId?: string;
  researchConsolidationProfileId?: string;
  researchAnalysisProfileId?: string;
  embeddingProfileId?: string; // Added missing field for embedding profile ID
}
export interface Config {
  summarization?: SummarizationConfig;
  retrieval?: RetrievalConfig;
  preferences?: PreferencesConfig;
  webSearch?: WebSearchConfig;
  modelProfiles?: ModelProfilesConfig;
}

export function useConfig() {
  const [config, setConfig] = useState<Config | null>(null);
  const [userConfig, setUserConfig] = useState<Config | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const { user } = useAuth();

  // Fetch user configuration from the API
  const fetchConfig = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const token = getToken(user);
      if (!token) {
        throw new Error('Authentication required');
      }
      
      const data = await getConfig(token);
      setConfig(data);
      
      // Extract just the user-configurable parts into userConfig
      const extractedUserConfig: Config = {
        summarization: data.summarization,
        retrieval: data.retrieval,
        webSearch: data.webSearch,
        preferences: {
          theme: data.preferences?.theme || "light",
          fontSize: data.preferences?.fontSize || 14,
          notificationsOn: data.preferences?.notificationsOn !== false,
          language: data.preferences?.language || "en"
        },
        modelProfiles: data.modelProfiles
      };
      
      setUserConfig(extractedUserConfig);
    } catch (err) {
      console.error('Error fetching configuration:', err);
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setIsLoading(false);
    }
  }, [user]);

  // Update full user configuration
  const updateUserConfig = async (newConfig: Config): Promise<boolean> => {
    if (!getToken(user)) {
      setError(new Error('Authentication required'));
      return false;
    }
    
    try {
      await updateConfig(getToken(user), newConfig);
      // Refresh the config after update
      await fetchConfig();
      return true;
    } catch (err) {
      console.error('Error updating configuration:', err);
      setError(err instanceof Error ? err : new Error(String(err)));
      return false;
    }
  };

  // Update a section of the user configuration (e.g., just summarization settings)
  const updatePartialConfig =  async (section: keyof Config, sectionConfig: unknown): Promise<boolean> => {
    if (!userConfig) {
      setError(new Error('No existing configuration to update'));
      return false;
    }
    
    // Create a copy with the updated section
    const updatedConfig = {
      ...userConfig,
      [section]: sectionConfig
    };
    
    return await updateUserConfig(updatedConfig);
  };

  // Load configuration on mount and when user changes
  useEffect(() => {
    if (getToken(user)) {
      fetchConfig();
    }
  }, [user, fetchConfig]);

  return { 
    config, 
    userConfig, 
    isLoading, 
    error,
    fetchConfig,
    updateConfig: setConfig,
    updatePartialConfig
  };
}