import React, { useEffect, useState } from 'react';
import { Job, ProgressUpdate } from '../types';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { Activity, AlertCircle } from 'lucide-react';

interface Props {
  job: Job;
}

interface SpeedDataPoint {
  time: string;
  speed: number;
}

export const ProgressMonitor: React.FC<Props> = ({ job }) => {
  const [speedHistory, setSpeedHistory] = useState<SpeedDataPoint[]>([]);
  const [wsConnected, setWsConnected] = useState(false);

  useEffect(() => {
    if (job.status !== 'running') {
      return;
    }

    // Connect to WebSocket for real-time updates
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.hostname}:8080/api/ws/jobs/${job.id}`;
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('WebSocket connected');
      setWsConnected(true);
    };

    ws.onmessage = (event) => {
      try {
        const data: ProgressUpdate = JSON.parse(event.data);
        if (data.type === 'progress') {
          // Add to speed history
          const now = new Date();
          const timeStr = now.toLocaleTimeString();
          setSpeedHistory((prev) => {
            const newHistory = [
              ...prev,
              {
                time: timeStr,
                speed: data.speed / (1024 * 1024), // Convert to MB/s
              },
            ];
            // Keep only last 20 points
            return newHistory.slice(-20);
          });
        }
      } catch (err) {
        console.error('Error parsing WebSocket message:', err);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      setWsConnected(false);
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected');
      setWsConnected(false);
    };

    return () => {
      ws.close();
    };
  }, [job.id, job.status]);

  const formatBytes = (bytes: number): string => {
    const gb = bytes / (1024 * 1024 * 1024);
    return `${gb.toFixed(2)} GB`;
  };

  const formatSpeed = (bytesPerSec: number): string => {
    const mbps = bytesPerSec / (1024 * 1024);
    return `${mbps.toFixed(2)} MB/s`;
  };

  return (
    <div className="space-y-6">
      {/* Connection Status */}
      <div className="flex items-center gap-2">
        <div className={`w-3 h-3 rounded-full ${wsConnected ? 'bg-green-500' : 'bg-gray-300'}`} />
        <span className="text-sm text-gray-600">
          {wsConnected ? 'Real-time updates active' : 'Polling for updates'}
        </span>
      </div>

      {/* Job Info */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <InfoCard label="Status" value={job.status_str} />
        <InfoCard label="Phase" value={job.phase} />
        <InfoCard label="Progress" value={`${job.progress.toFixed(1)}%`} />
        <InfoCard label="Speed" value={formatSpeed(job.speed)} />
      </div>

      {/* Progress Details */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-gray-50 rounded-lg p-4">
          <p className="text-sm text-gray-600 mb-1">Bytes Processed</p>
          <p className="text-2xl font-bold text-gray-900">
            {formatBytes(job.bytes_done)}
          </p>
          <p className="text-xs text-gray-500 mt-1">
            of {formatBytes(job.bytes_total)}
          </p>
        </div>

        <div className="bg-gray-50 rounded-lg p-4">
          <p className="text-sm text-gray-600 mb-1">Bad Sectors</p>
          <p className="text-2xl font-bold text-gray-900">{job.bad_sectors}</p>
          <p className="text-xs text-gray-500 mt-1">
            {job.bad_sectors > 0 ? 'Errors detected' : 'No errors'}
          </p>
        </div>

        <div className="bg-gray-50 rounded-lg p-4">
          <p className="text-sm text-gray-600 mb-1">ETA</p>
          <p className="text-2xl font-bold text-gray-900">
            {job.eta || 'Calculating...'}
          </p>
          <p className="text-xs text-gray-500 mt-1">Estimated time</p>
        </div>
      </div>

      {/* Speed Graph */}
      {speedHistory.length > 0 && (
        <div className="bg-white border border-gray-200 rounded-lg p-4">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
            <Activity size={20} className="text-forensic-600" />
            Transfer Speed
          </h3>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={speedHistory}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="time"
                tick={{ fontSize: 12 }}
                angle={-45}
                textAnchor="end"
                height={60}
              />
              <YAxis
                label={{ value: 'MB/s', angle: -90, position: 'insideLeft' }}
                tick={{ fontSize: 12 }}
              />
              <Tooltip
                formatter={(value: number) => `${value.toFixed(2)} MB/s`}
              />
              <Line
                type="monotone"
                dataKey="speed"
                stroke="#0ea5e9"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}

      {/* Errors */}
      {job.errors && job.errors.length > 0 && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <h3 className="text-lg font-semibold mb-2 flex items-center gap-2 text-red-800">
            <AlertCircle size={20} />
            Errors
          </h3>
          <div className="space-y-1">
            {job.errors.map((error, index) => (
              <p key={index} className="text-sm text-red-700">
                {error}
              </p>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

interface InfoCardProps {
  label: string;
  value: string;
}

const InfoCard: React.FC<InfoCardProps> = ({ label, value }) => {
  return (
    <div className="bg-gray-50 rounded-lg p-3">
      <p className="text-xs text-gray-600 mb-1">{label}</p>
      <p className="text-lg font-semibold text-gray-900">{value}</p>
    </div>
  );
};
