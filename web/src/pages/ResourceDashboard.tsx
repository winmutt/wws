import React, { useState, useEffect } from 'react';

interface UsageData {
  workspace_id: number;
  cpu_usage: number;
  memory_usage: number;
  storage_used_gb: number;
  network_in_mb: number;
  network_out_mb: number;
  timestamp: string;
}

interface Alert {
  id: number;
  workspace_id: number;
  alert_type: string;
  severity: string;
  message: string;
  value: number;
  threshold: number;
  acknowledged: boolean;
  created_at: string;
}

interface WorkspaceMetrics {
  id: number;
  name: string;
  tag: string;
  status: string;
  cpuUsage: number;
  memoryUsage: number;
  storageGB: number;
  uptime: string;
}

function ResourceDashboard() {
  const [metrics, setMetrics] = useState<WorkspaceMetrics[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [summary, setSummary] = useState({
    totalWorkspaces: 0,
    highCPUCount: 0,
    highMemoryCount: 0,
    activeAlerts: 0,
  });
  const [isLoading, setIsLoading] = useState(true);
  const [refreshInterval, setRefreshInterval] = useState(30000); // 30 seconds

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, refreshInterval);
    return () => clearInterval(interval);
  }, [refreshInterval]);

  const loadData = async () => {
    setIsLoading(true);
    try {
      const [usageData, activeAlerts] = await Promise.all([
        fetch('/api/v1/analytics/usage').then(r => r.json()),
        fetch('/api/v1/analytics/alerts').then(r => r.json()),
      ]);

      // Process metrics
      const processedMetrics: WorkspaceMetrics[] = usageData.map((u: UsageData) => ({
        id: u.workspace_id,
        name: `Workspace ${u.workspace_id}`,
        tag: `ws-${u.workspace_id}`,
        status: u.cpu_usage > 80 ? 'high-load' : 'running',
        cpuUsage: u.cpu_usage,
        memoryUsage: u.memory_usage,
        storageGB: u.storage_used_gb,
        uptime: calculateUptime(u.timestamp),
      }));

      setMetrics(processedMetrics);
      setAlerts(activeAlerts || []);

      // Update summary
      setSummary({
        totalWorkspaces: processedMetrics.length,
        highCPUCount: processedMetrics.filter(m => m.cpuUsage > 80).length,
        highMemoryCount: processedMetrics.filter(m => m.memoryUsage > 80).length,
        activeAlerts: activeAlerts?.filter((a: Alert) => !a.acknowledged).length || 0,
      });
    } catch (error) {
      console.error('Failed to load resource metrics:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const calculateUptime = (timestamp: string): string => {
    const start = new Date(timestamp);
    const now = new Date();
    const diff = Math.floor((now.getTime() - start.getTime()) / 1000);
    
    if (diff < 60) return `${diff}s`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h`;
    return `${Math.floor(diff / 86400)}d`;
  };

  const acknowledgeAlert = async (alertId: number) => {
    try {
      await fetch('/api/v1/analytics/alerts/resolve', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ alert_id: alertId }),
      });
      loadData();
    } catch (error) {
      console.error('Failed to acknowledge alert:', error);
    }
  };

  const getSeverityColor = (severity: string): string => {
    switch (severity.toLowerCase()) {
      case 'critical': return 'bg-red-100 text-red-800';
      case 'high': return 'bg-orange-100 text-orange-800';
      case 'medium': return 'bg-yellow-100 text-yellow-800';
      case 'low': return 'bg-blue-100 text-blue-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getUsageColor = (usage: number): string => {
    if (usage > 90) return 'bg-red-500';
    if (usage > 70) return 'bg-orange-500';
    if (usage > 50) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  if (isLoading) {
    return (
      <div className="p-4">
        <div className="text-center py-12">
          <p className="text-gray-600">Loading resource dashboard...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">Resource Monitoring</h2>
        <div className="flex items-center space-x-4">
          <select
            value={refreshInterval}
            onChange={(e) => setRefreshInterval(Number(e.target.value))}
            className="border rounded-md px-3 py-2"
          >
            <option value={15000}>15s refresh</option>
            <option value={30000}>30s refresh</option>
            <option value={60000}>1min refresh</option>
            <option value={300000}>5min refresh</option>
          </select>
          <button
            onClick={loadData}
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-sm font-medium text-gray-500 mb-2">Total Workspaces</h3>
          <p className="text-3xl font-bold text-gray-900">{summary.totalWorkspaces}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-sm font-medium text-gray-500 mb-2">High CPU Usage</h3>
          <p className="text-3xl font-bold text-orange-600">{summary.highCPUCount}</p>
          <p className="text-xs text-gray-500 mt-2">&gt;80% CPU utilization</p>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-sm font-medium text-gray-500 mb-2">High Memory Usage</h3>
          <p className="text-3xl font-bold text-orange-600">{summary.highMemoryCount}</p>
          <p className="text-xs text-gray-500 mt-2">&gt;80% memory utilization</p>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-sm font-medium text-gray-500 mb-2">Active Alerts</h3>
          <p className="text-3xl font-bold text-red-600">{summary.activeAlerts}</p>
          <p className="text-xs text-gray-500 mt-2">Requires attention</p>
        </div>
      </div>

      {/* Active Alerts */}
      {alerts.filter((a: Alert) => !a.acknowledged).length > 0 && (
        <div className="bg-white rounded-lg shadow mb-8">
          <div className="p-6 border-b border-gray-200">
            <h3 className="text-lg font-semibold">Active Alerts</h3>
          </div>
          <div className="p-6">
            <div className="space-y-4">
              {alerts
                .filter((a: Alert) => !a.acknowledged)
                .map((alert: Alert) => (
                  <div key={alert.id} className="flex justify-between items-center p-4 bg-gray-50 rounded-lg">
                    <div className="flex-1">
                      <div className="flex items-center space-x-3">
                        <span className={`px-2 py-1 rounded-full text-xs font-medium ${getSeverityColor(alert.severity)}`}>
                          {alert.severity.toUpperCase()}
                        </span>
                        <span className="font-medium">{alert.alert_type}</span>
                      </div>
                      <p className="text-sm text-gray-600 mt-1">{alert.message}</p>
                      <p className="text-xs text-gray-500 mt-1">
                        Value: {alert.value} | Threshold: {alert.threshold}
                      </p>
                    </div>
                    <button
                      onClick={() => acknowledgeAlert(alert.id)}
                      className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
                    >
                      Acknowledge
                    </button>
                  </div>
                ))}
            </div>
          </div>
        </div>
      )}

      {/* Workspace Metrics */}
      <div className="bg-white rounded-lg shadow">
        <div className="p-6 border-b border-gray-200">
          <h3 className="text-lg font-semibold">Workspace Resource Usage</h3>
        </div>
        <div className="p-6">
          {metrics.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-gray-600">No workspaces with metrics data</p>
            </div>
          ) : (
            <div className="space-y-6">
              {metrics.map((metric) => (
                <div key={metric.id} className="border rounded-lg p-4">
                  <div className="flex justify-between items-start mb-4">
                    <div>
                      <h4 className="font-medium">{metric.name}</h4>
                      <p className="text-sm text-gray-500">{metric.tag}</p>
                    </div>
                    <div className="text-right">
                      <span className="text-xs text-gray-500">Uptime</span>
                      <p className="font-medium">{metric.uptime}</p>
                    </div>
                  </div>

                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <div className="flex justify-between mb-1">
                        <span className="text-sm text-gray-600">CPU Usage</span>
                        <span className="text-sm font-medium">{metric.cpuUsage}%</span>
                      </div>
                      <div className="w-full bg-gray-200 rounded-full h-2">
                        <div
                          className={`h-2 rounded-full ${getUsageColor(metric.cpuUsage)}`}
                          style={{ width: `${metric.cpuUsage}%` }}
                        ></div>
                      </div>
                    </div>

                    <div>
                      <div className="flex justify-between mb-1">
                        <span className="text-sm text-gray-600">Memory Usage</span>
                        <span className="text-sm font-medium">{metric.memoryUsage}%</span>
                      </div>
                      <div className="w-full bg-gray-200 rounded-full h-2">
                        <div
                          className={`h-2 rounded-full ${getUsageColor(metric.memoryUsage)}`}
                          style={{ width: `${metric.memoryUsage}%` }}
                        ></div>
                      </div>
                    </div>

                    <div>
                      <div className="flex justify-between mb-1">
                        <span className="text-sm text-gray-600">Storage</span>
                        <span className="text-sm font-medium">{metric.storageGB} GB</span>
                      </div>
                      <div className="w-full bg-gray-200 rounded-full h-2">
                        <div
                          className="h-2 rounded-full bg-blue-500"
                          style={{ width: `${Math.min(metric.storageGB, 100)}%` }}
                        ></div>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default ResourceDashboard;
