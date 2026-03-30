import { create } from 'zustand';
import { Job, DiskInfo } from '../types';

interface JobStore {
  jobs: Job[];
  disks: DiskInfo[];
  selectedJob: Job | null;

  setJobs: (jobs: Job[]) => void;
  addJob: (job: Job) => void;
  updateJob: (jobId: string, updates: Partial<Job>) => void;
  removeJob: (jobId: string) => void;
  setSelectedJob: (job: Job | null) => void;

  setDisks: (disks: DiskInfo[]) => void;
}

export const useJobStore = create<JobStore>((set) => ({
  jobs: [],
  disks: [],
  selectedJob: null,

  setJobs: (jobs) => set({ jobs }),

  addJob: (job) => set((state) => ({
    jobs: [...state.jobs, job]
  })),

  updateJob: (jobId, updates) => set((state) => ({
    jobs: state.jobs.map(job =>
      job.id === jobId ? { ...job, ...updates } : job
    ),
    selectedJob: state.selectedJob?.id === jobId
      ? { ...state.selectedJob, ...updates }
      : state.selectedJob
  })),

  removeJob: (jobId) => set((state) => ({
    jobs: state.jobs.filter(job => job.id !== jobId),
    selectedJob: state.selectedJob?.id === jobId ? null : state.selectedJob
  })),

  setSelectedJob: (job) => set({ selectedJob: job }),

  setDisks: (disks) => set({ disks }),
}));
