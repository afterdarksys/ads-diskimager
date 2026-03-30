import React, { useEffect } from 'react';
import { useJobStore } from '../store/jobStore';
import { HardDrive, Loader } from 'lucide-react';
import axios from 'axios';

export const DiskSelector: React.FC = () => {
  const { disks, setDisks } = useJobStore();
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  useEffect(() => {
    fetchDisks();
  }, []);

  const fetchDisks = async () => {
    try {
      setLoading(true);
      const response = await axios.get('/api/disks');
      setDisks(response.data);
      setError(null);
    } catch (err) {
      setError('Failed to load disks');
      console.error('Error fetching disks:', err);
    } finally {
      setLoading(false);
    }
  };

  const formatSize = (bytes: number): string => {
    const gb = bytes / (1024 * 1024 * 1024);
    if (gb >= 1024) {
      return `${(gb / 1024).toFixed(1)} TB`;
    }
    return `${gb.toFixed(1)} GB`;
  };

  const getHealthColor = (health: string): string => {
    return health === 'Healthy' ? 'text-green-600' : 'text-red-600';
  };

  const getTypeIcon = (type: string): React.ReactNode => {
    return <HardDrive className="text-gray-600" size={20} />;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader className="animate-spin text-forensic-600" size={32} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-8">
        <p className="text-red-600 mb-2">{error}</p>
        <button
          onClick={fetchDisks}
          className="px-4 py-2 bg-forensic-600 text-white rounded hover:bg-forensic-700 transition"
        >
          Retry
        </button>
      </div>
    );
  }

  if (disks.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500">
        <HardDrive className="mx-auto mb-2" size={48} />
        <p>No disks available</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {disks.map((disk) => (
        <div
          key={disk.path}
          className="border border-gray-200 rounded-lg p-4 hover:border-forensic-500 hover:shadow-md transition cursor-pointer"
        >
          <div className="flex items-start gap-3">
            {getTypeIcon(disk.type)}
            <div className="flex-1 min-w-0">
              <h3 className="font-semibold text-gray-900 truncate">{disk.name}</h3>
              <p className="text-sm text-gray-600 truncate">{disk.path}</p>
              <div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
                <span>{formatSize(disk.size)}</span>
                <span className="truncate">{disk.model}</span>
              </div>
              <div className="mt-1 flex items-center gap-2">
                <span className={`text-xs font-medium ${getHealthColor(disk.health)}`}>
                  {disk.health}
                </span>
                <span className="text-xs text-gray-400">•</span>
                <span className="text-xs text-gray-500">{disk.type}</span>
              </div>
            </div>
          </div>
        </div>
      ))}

      <button
        onClick={fetchDisks}
        className="w-full py-2 text-sm text-forensic-600 hover:text-forensic-700 font-medium"
      >
        Refresh
      </button>
    </div>
  );
};
