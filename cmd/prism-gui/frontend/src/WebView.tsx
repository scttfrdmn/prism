import React, { useState, useEffect } from 'react';
import { Alert, Box, Button, SpaceBetween, Spinner } from '@cloudscape-design/components';

interface WebViewProps {
  url: string;
  serviceName: string;
  instanceName: string;
}

const WebView: React.FC<WebViewProps> = ({ url, serviceName, instanceName }) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState(Date.now());

  useEffect(() => {
    // Reset loading state when URL changes
    setLoading(true);
    setError(null);
    setLastRefresh(Date.now());
  }, [url]);

  const handleIframeLoad = () => {
    setLoading(false);
    setError(null);
  };

  const handleIframeError = () => {
    setLoading(false);
    setError('Failed to load web service. Please check if the service is running and accessible.');
  };

  const handleRefresh = () => {
    setLoading(true);
    setError(null);
    setLastRefresh(Date.now());
  };

  const handleOpenInBrowser = () => {
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Header with controls */}
      <div
        style={{
          padding: '12px 16px',
          backgroundColor: '#232f3e',
          color: 'white',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '1px solid #414d5c',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <span style={{ fontWeight: 'bold', fontSize: '14px' }}>
            {serviceName} - {instanceName}
          </span>
          {loading && (
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <Spinner size="small" />
              <span style={{ fontSize: '12px', opacity: 0.8 }}>Loading...</span>
            </div>
          )}
        </div>
        <div style={{ display: 'flex', gap: '8px' }}>
          <Button onClick={handleRefresh} iconName="refresh" variant="icon">
            Refresh
          </Button>
          <Button onClick={handleOpenInBrowser} iconName="external" variant="icon">
            Open in Browser
          </Button>
        </div>
      </div>

      {/* URL bar */}
      <div
        style={{
          padding: '8px 16px',
          backgroundColor: '#f2f3f3',
          borderBottom: '1px solid #d5dbdb',
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
        }}
      >
        <span style={{ fontSize: '11px', color: '#5f6b7a', fontWeight: 'bold' }}>URL:</span>
        <span
          style={{
            fontSize: '12px',
            fontFamily: 'monospace',
            color: '#0972d3',
            flex: 1,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {url}
        </span>
      </div>

      {/* Error display */}
      {error && (
        <div style={{ padding: '16px' }}>
          <Alert type="error" header="Connection Error">
            <SpaceBetween size="s">
              <Box>{error}</Box>
              <Box>
                <strong>Troubleshooting:</strong>
                <ul style={{ marginTop: '8px', marginLeft: '20px' }}>
                  <li>Verify the service is running on the instance</li>
                  <li>Check if the instance is accessible via SSH</li>
                  <li>Ensure the correct port is open in security groups</li>
                  <li>Try opening in external browser to check connectivity</li>
                </ul>
              </Box>
            </SpaceBetween>
          </Alert>
        </div>
      )}

      {/* iframe container */}
      <div
        style={{
          flex: 1,
          position: 'relative',
          backgroundColor: '#ffffff',
          overflow: 'hidden',
        }}
      >
        <iframe
          key={lastRefresh} // Force reload on refresh
          src={url}
          style={{
            width: '100%',
            height: '100%',
            border: 'none',
            display: error ? 'none' : 'block',
          }}
          title={`${serviceName} - ${instanceName}`}
          onLoad={handleIframeLoad}
          onError={handleIframeError}
          sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-modals"
          allow="clipboard-read; clipboard-write"
        />
        {loading && !error && (
          <div
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              backgroundColor: 'rgba(255, 255, 255, 0.95)',
              gap: '16px',
            }}
          >
            <Spinner size="large" />
            <Box variant="p" color="text-body-secondary">
              Loading {serviceName}...
            </Box>
            <Box variant="small" color="text-body-secondary">
              This may take a few moments
            </Box>
          </div>
        )}
      </div>

      {/* Help footer */}
      <div
        style={{
          padding: '8px 16px',
          backgroundColor: '#f2f3f3',
          borderTop: '1px solid #d5dbdb',
          fontSize: '11px',
          color: '#5f6b7a',
        }}
      >
        <strong>💡 Tip:</strong> If you experience issues, try opening in an external browser using the button above.
        Some services may require additional authentication or configuration.
      </div>
    </div>
  );
};

export default WebView;
