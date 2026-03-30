export interface DiskInfo {
  path: string;
  name: string;
  size: number;
  model: string;
  serial: string;
  health: string;
  type: 'HDD' | 'SSD' | 'USB';
}

export interface Job {
  id: string;
  type: 'image' | 'restore';
  status: 'pending' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled';
  phase: string;
  progress: number;
  speed: number;
  eta: string;
  bytes_total: number;
  bytes_done: number;
  bad_sectors: number;
  errors: string[];
  created_at: string;
  completed_at?: string;
  source_path: string;
  dest_path: string;
  format: string;
  hash?: string;
  status_str: string;
}

export interface ProgressUpdate {
  type: 'progress';
  job_id: string;
  bytes_processed: number;
  total_bytes: number;
  phase: string;
  message: string;
  speed: number;
  eta: number;
  percentage: number;
  bad_sectors: number;
  errors: string[];
  timestamp: string;
}

export interface JobRequest {
  type: 'image' | 'restore';
  source_path: string;
  dest_path: string;
  format: string;
  compression: number;
  metadata: {
    case_number?: string;
    examiner?: string;
    evidence?: string;
    description?: string;
  };
}
