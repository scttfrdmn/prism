import React, { useEffect, useRef, useState } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';

interface TerminalProps {
  instanceName: string;
}

const Terminal: React.FC<TerminalProps> = React.memo(({ instanceName }) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!terminalRef.current || !instanceName) return;

    // Prevent recreating terminal if instance hasn't changed
    if (xtermRef.current && wsRef.current?.readyState === WebSocket.OPEN) {
      console.log('Terminal already connected, skipping reconnect');
      return;
    }

    // Create terminal
    const term = new XTerm({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
        cursor: '#aeafad',
        black: '#000000',
        red: '#cd3131',
        green: '#0dbc79',
        yellow: '#e5e510',
        blue: '#2472c8',
        magenta: '#bc3fbc',
        cyan: '#11a8cd',
        white: '#e5e5e5',
        brightBlack: '#666666',
        brightRed: '#f14c4c',
        brightGreen: '#23d18b',
        brightYellow: '#f5f543',
        brightBlue: '#3b8eea',
        brightMagenta: '#d670d6',
        brightCyan: '#29b8db',
        brightWhite: '#ffffff',
      },
      cols: 80,
      rows: 24,
    });

    // Add addons
    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();

    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);

    // Open terminal in DOM
    term.open(terminalRef.current);
    fitAddon.fit();

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    // Connect to WebSocket
    const wsURL = `ws://localhost:8948/terminal?instance=${encodeURIComponent(instanceName)}`;
    const ws = new WebSocket(wsURL);

    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      console.log('WebSocket connected');
      setIsConnected(true);
      setError(null);

      // Send terminal size
      const size = {
        type: 'resize',
        size: {
          rows: term.rows,
          cols: term.cols,
        },
      };
      ws.send(JSON.stringify(size));
    };

    ws.onmessage = (event) => {
      // Handle incoming data from SSH
      if (event.data instanceof ArrayBuffer) {
        const uint8Array = new Uint8Array(event.data);
        term.write(uint8Array);
      } else if (typeof event.data === 'string') {
        term.write(event.data);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      setError('Connection error - check if workspace is running');
      setIsConnected(false);
    };

    ws.onclose = () => {
      console.log('WebSocket closed');
      setIsConnected(false);
      term.write('\r\n\x1b[31mConnection closed\x1b[0m\r\n');
    };

    wsRef.current = ws;

    // Handle terminal input
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        const message = {
          type: 'input',
          data: data,
        };
        ws.send(JSON.stringify(message));
      }
    });

    // Handle terminal resize
    const handleResize = () => {
      if (fitAddon && ws.readyState === WebSocket.OPEN) {
        fitAddon.fit();
        const size = {
          type: 'resize',
          size: {
            rows: term.rows,
            cols: term.cols,
          },
        };
        ws.send(JSON.stringify(size));
      }
    };

    window.addEventListener('resize', handleResize);

    // Cleanup
    return () => {
      window.removeEventListener('resize', handleResize);
      ws.close();
      term.dispose();
    };
  }, [instanceName]);

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Connection status bar */}
      <div
        style={{
          padding: '8px 16px',
          backgroundColor: isConnected ? '#0dbc79' : '#cd3131',
          color: 'white',
          fontSize: '12px',
          fontWeight: 'bold',
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
        }}
      >
        <span style={{
          width: '8px',
          height: '8px',
          borderRadius: '50%',
          backgroundColor: 'white',
          animation: isConnected ? 'pulse 2s infinite' : 'none'
        }} />
        {isConnected ? (
          `Connected to ${instanceName}`
        ) : error ? (
          `Error: ${error}`
        ) : (
          'Connecting...'
        )}
      </div>

      {/* Terminal container */}
      <div
        ref={terminalRef}
        style={{
          flex: 1,
          padding: '10px',
          backgroundColor: '#1e1e1e',
          overflow: 'hidden',
        }}
      />

      {/* Add pulse animation */}
      <style>{`
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.5; }
        }
      `}</style>
    </div>
  );
});

export default Terminal;
