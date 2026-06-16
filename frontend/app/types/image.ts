export interface ImageData {
  filename: string
  status: 'success' | 'error'
  message?: string
  format?: string
  orientation?: string
  expiryTime?: string
  urls?: {
    original: string
    webp: string
    thumb?: string
    avif?: string
  }
  tags?: string[]
  id?: string
  path?: string
}

export interface CopyStatus {
  type: string
} 
