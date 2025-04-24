import { getHeaders, req } from "./base"
import { Model } from "./types"

export const getModels = async (accessToken: string) =>
  await req<{ models: Model[] }>({
    method: 'GET',
    headers: getHeaders(accessToken),
    path: 'api/models'
  })