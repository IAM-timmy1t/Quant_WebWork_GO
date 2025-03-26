/**
 * MetricsChart.tsx
 * 
 * @module components/charts
 * @description Reusable chart component for displaying time series metrics data
 * @version 1.0.0
 */

import React, { useEffect, useRef } from 'react';
import Chart from 'chart.js/auto';
import { get } from 'lodash';

interface MetricsChartProps {
  // Data array containing time series data points
  data: Array<{
    timestamp: string | number;
    [key: string]: any;
  }>;
  
  // Array of metric keys to plot from each data point
  metrics: string[];
  
  // Display labels for each metric
  labels: string[];
  
  // Colors for each metric line
  colors?: string[];
  
  // Height of the chart in pixels
  height?: number;
  
  // Function to format Y-axis values
  formatY?: (value: number) => string;
  
  // Function to format timestamps
  formatTimestamp?: (timestamp: string | number) => string;
  
  // Chart title
  title?: string;
}

export const MetricsChart: React.FC<MetricsChartProps> = ({
  data,
  metrics,
  labels,
  colors = ['#1890ff', '#52c41a', '#722ed1', '#faad14', '#f5222d'],
  height = 300,
  formatY = (value: number) => value.toString(),
  formatTimestamp = (timestamp: string | number) => {
    if (typeof timestamp === 'string') {
      return new Date(timestamp).toLocaleTimeString();
    }
    return new Date(timestamp).toLocaleTimeString();
  },
  title,
}) => {
  const chartRef = useRef<HTMLCanvasElement>(null);
  const chartInstance = useRef<Chart | null>(null);
  
  // Create or update chart when data changes
  useEffect(() => {
    // Skip if no data or DOM element not ready
    if (!data || !data.length || !chartRef.current) {
      return;
    }
    
    // Clean up existing chart instance if it exists
    if (chartInstance.current) {
      chartInstance.current.destroy();
    }
    
    // Extract timestamps for x-axis
    const timestamps = data.map(d => formatTimestamp(d.timestamp));
    
    // Prepare datasets
    const datasets = metrics.map((metric, index) => {
      return {
        label: labels[index] || metric,
        data: data.map(d => get(d, metric, 0)),
        borderColor: colors[index % colors.length],
        backgroundColor: `${colors[index % colors.length]}33`, // Add transparency
        fill: false,
        tension: 0.3, // Slight curve for lines
        pointRadius: 2,
        pointHoverRadius: 5,
      };
    });
    
    // Create new chart
    const ctx = chartRef.current.getContext('2d');
    if (ctx) {
      chartInstance.current = new Chart(ctx, {
        type: 'line',
        data: {
          labels: timestamps,
          datasets,
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          interaction: {
            mode: 'index',
            intersect: false,
          },
          plugins: {
            title: {
              display: !!title,
              text: title || '',
              font: {
                size: 16,
              },
            },
            tooltip: {
              enabled: true,
              callbacks: {
                label: (context) => {
                  const value = context.raw as number;
                  return `${context.dataset.label}: ${formatY(value)}`;
                },
              },
            },
            legend: {
              position: 'top',
              labels: {
                usePointStyle: true,
                boxWidth: 6,
              },
            },
          },
          scales: {
            x: {
              grid: {
                display: false,
              },
              ticks: {
                maxRotation: 0,
                autoSkip: true,
                maxTicksLimit: 8,
              },
            },
            y: {
              beginAtZero: true,
              ticks: {
                callback: function(value) {
                  return formatY(value as number);
                },
              },
            },
          },
        },
      });
    }
    
    // Cleanup on unmount
    return () => {
      if (chartInstance.current) {
        chartInstance.current.destroy();
        chartInstance.current = null;
      }
    };
  }, [data, metrics, labels, colors, formatY, formatTimestamp, title]);
  
  return (
    <div className="metrics-chart-container" style={{ height }}>
      {data && data.length > 0 ? (
        <canvas ref={chartRef} />
      ) : (
        <div className="no-data-message">
          <p>No data available</p>
        </div>
      )}
    </div>
  );
}; 