describe('Bridge Module Integration', () => {
  beforeEach(() => {
    // Mock server responses for testing
    cy.intercept('POST', 'http://localhost:8080/bridge-service/v1**', (req) => {
      // Parse the request body
      const body = req.body;
      
      // Match different request types
      if (body.type === 'analysis-request') {
        req.reply({
          statusCode: 200,
          body: {
            id: 'test-response-id',
            type: 'analysis-response',
            content: '',
            metadata: {
              token_address: body.metadata?.token_address,
              analysis: {
                risk_score: 0.25,
                risk_level: 'low',
                confidence: 0.95,
                warnings: [],
                insights: [
                  { type: 'liquidity', value: 'high', description: 'Token has high liquidity' },
                  { type: 'volatility', value: 'low', description: 'Token has low volatility' }
                ]
              },
              timestamp: Date.now()
            },
            timestamp: Date.now()
          }
        });
      } else if (body.type === 'metrics') {
        req.reply({
          statusCode: 200,
          body: {
            id: 'metrics-response-id',
            type: 'metrics',
            content: '',
            metadata: {
              metrics: {
                totalMessages: 1024,
                activeConnections: 8,
                messagesPerSecond: 42.5,
                averageProcessingTimeMs: 15.2,
                totalErrors: 2,
                uptime: '1d 6h 32m'
              }
            },
            timestamp: Date.now()
          }
        });
      } else {
        // Generic response for other message types
        req.reply({
          statusCode: 200,
          body: {
            id: 'generic-response-id',
            type: `${body.type}-response`,
            content: 'Response content',
            metadata: { status: 'success' },
            timestamp: Date.now()
          }
        });
      }
    }).as('bridgeRequest');

    // Visit the application
    cy.visit('/');
  });

  it('renders the bridge connection tab correctly', () => {
    // Should be on the Bridge Connection tab by default
    cy.contains('h2', 'Bridge Connection').should('be.visible');
    cy.contains('button', 'Connect').should('be.visible');
    
    // Connection status should show as disconnected
    cy.contains('Connection Status').next().contains('Disconnected').should('be.visible');
  });

  it('navigates to bridge verification tab', () => {
    // Click on the verification tab
    cy.contains('button', 'Bridge Verification').click();
    
    // Should now show the verification tab
    cy.contains('h2', 'Bridge Module Verification').should('be.visible');
    cy.contains('button', 'Run Verification').should('be.visible');
  });

  it('connects to the bridge and displays connection status', () => {
    // Connect to the bridge
    cy.contains('button', 'Connect').click();
    
    // Connection status should update
    cy.contains('Connection Status').next().contains('Connected').should('be.visible');
    
    // Button should change to Disconnect
    cy.contains('button', 'Disconnect').should('be.visible');
  });

  it('sends a message and displays the response', () => {
    // Connect to the bridge
    cy.contains('button', 'Connect').click();
    
    // Wait for connection
    cy.contains('Connection Status').next().contains('Connected').should('be.visible');
    
    // Enter message
    cy.get('#message-input').type('Test message');
    
    // Select message type
    cy.get('#message-type').select('query');
    
    // Send message
    cy.contains('button', 'Send Message').click();
    
    // Wait for response
    cy.wait('@bridgeRequest');
    
    // Response should be displayed
    cy.contains('Response:').should('be.visible');
    cy.contains('query-response').should('be.visible');
  });

  it('analyzes a token and displays results', () => {
    // Connect to the bridge
    cy.contains('button', 'Connect').click();
    
    // Wait for connection
    cy.contains('Connection Status').next().contains('Connected').should('be.visible');
    
    // Enter token address
    cy.get('#token-address').type('0x1234567890abcdef1234567890abcdef12345678');
    
    // Analyze token
    cy.contains('button', 'Analyze Token').click();
    
    // Wait for response
    cy.wait('@bridgeRequest');
    
    // Analysis result should be displayed
    cy.contains('Analysis Result:').should('be.visible');
    cy.contains('Risk Level: low').should('be.visible');
  });

  it('runs verification and displays test results', () => {
    // Navigate to verification tab
    cy.contains('button', 'Bridge Verification').click();
    
    // Run verification
    cy.contains('button', 'Run Verification').click();
    
    // Wait for tests to complete
    cy.wait('@bridgeRequest').wait('@bridgeRequest').wait('@bridgeRequest');
    
    // Test results should be displayed
    cy.contains('Test Results').should('be.visible');
    cy.contains('Connection').should('be.visible');
    cy.contains('Metrics Endpoint').should('be.visible');
    cy.contains('Token Analysis').should('be.visible');
    
    // At least one successful test
    cy.contains('SUCCESS').should('be.visible');
  });

  it('handles errors gracefully', () => {
    // Intercept and force an error for this test only
    cy.intercept('POST', 'http://localhost:8080/bridge-service/v1**', {
      statusCode: 500,
      body: {
        error: 'Internal server error'
      }
    }).as('errorRequest');
    
    // Connect to the bridge
    cy.contains('button', 'Connect').click();
    
    // It should show an error
    cy.contains('Error:').should('be.visible');
  });

  it('displays metrics when requested', () => {
    // Connect to the bridge
    cy.contains('button', 'Connect').click();
    
    // Wait for connection
    cy.contains('Connection Status').next().contains('Connected').should('be.visible');
    
    // Request metrics
    cy.contains('button', 'Get Metrics').click();
    
    // Wait for response
    cy.wait('@bridgeRequest');
    
    // Metrics should be displayed
    cy.contains('Bridge Metrics:').should('be.visible');
    cy.contains('Total Messages:').should('be.visible');
    cy.contains('Active Connections:').should('be.visible');
  });
});
