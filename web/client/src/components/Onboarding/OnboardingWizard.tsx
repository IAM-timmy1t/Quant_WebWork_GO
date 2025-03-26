/**
 * OnboardingWizard.tsx
 * 
 * @module components/Onboarding
 * @description Step-by-step guide for new users setting up the system.
 * @version 1.0.0
 */

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Steps, Card, Alert, Progress } from '../ui';
import { SecurityCheck } from './SecurityCheck';
import { BridgeSetup } from './BridgeSetup';
import { AdminSetup } from './AdminSetup';
import { useConfig } from '../../hooks/useConfig';

interface StepProps {
  onNext: () => void;
  onBack: () => void;
  onSkip: () => void;
}

// Step components (simplified for brevity)
const WelcomeStep: React.FC<StepProps> = ({ onNext }) => (
  <Card className="welcome-step">
    <h3>Welcome to QUANT_WebWork_GO</h3>
    <p>
      This wizard will guide you through setting up your private network system.
      We'll help you configure security, create your first bridge connection,
      and ensure everything is working correctly.
    </p>
    <Button onClick={onNext} primary>Begin Setup</Button>
  </Card>
);

const SecurityStep: React.FC<StepProps> = ({ onNext, onBack, onSkip }) => {
  const { config, updateConfig } = useConfig();
  const [securityScore, setSecurityScore] = useState(0);
  
  // Logic to check and update security settings
  
  return (
    <Card className="security-step">
      <h3>Security Configuration</h3>
      <Alert type={securityScore < 70 ? "warning" : "success"}>
        Security Score: {securityScore}%
      </Alert>
      <SecurityCheck onScoreChange={setSecurityScore} />
      <div className="button-group">
        <Button onClick={onBack}>Back</Button>
        <Button onClick={onSkip}>Skip for Now</Button>
        <Button 
          onClick={onNext} 
          primary 
          disabled={securityScore < 50}
          title={securityScore < 50 ? "Please address critical security issues before continuing" : ""}
        >
          Next
        </Button>
      </div>
    </Card>
  );
};

const BridgeStep: React.FC<StepProps> = ({ onNext, onBack, onSkip }) => {
  const [serviceName, setServiceName] = useState('');
  const [serviceHost, setServiceHost] = useState('localhost');
  const [servicePort, setServicePort] = useState('8000');
  const [serviceProtocol, setServiceProtocol] = useState('http');
  
  const handleAddService = async () => {
    // Logic to register a service
    onNext();
  };
  
  return (
    <Card className="bridge-step">
      <h3>Connect Your First Service</h3>
      <p>Let's set up your first bridge connection to expose a service.</p>
      
      <BridgeSetup 
        serviceName={serviceName}
        serviceHost={serviceHost}
        servicePort={servicePort}
        serviceProtocol={serviceProtocol}
        onServiceNameChange={setServiceName}
        onServiceHostChange={setServiceHost}
        onServicePortChange={setServicePort}
        onServiceProtocolChange={setServiceProtocol}
      />
      
      <div className="button-group">
        <Button onClick={onBack}>Back</Button>
        <Button onClick={onSkip}>Skip for Now</Button>
        <Button 
          onClick={handleAddService} 
          primary
          disabled={!serviceName}
        >
          Next
        </Button>
      </div>
    </Card>
  );
};

const FinishStep: React.FC<StepProps> = ({ onNext }) => (
  <Card className="finish-step">
    <h3>Setup Complete!</h3>
    <p>
      Your QUANT_WebWork_GO system is now configured and ready to use.
      You can always adjust settings later through the dashboard.
    </p>
    <Button onClick={onNext} primary>Go to Dashboard</Button>
  </Card>
);

// Main Onboarding Wizard Component
export const OnboardingWizard: React.FC = () => {
  const [currentStep, setCurrentStep] = useState(0);
  const [isCompleted, setIsCompleted] = useState(false);
  const navigate = useNavigate();
  
  const steps = [
    {
      title: 'Welcome',
      content: (
        <WelcomeStep
          onNext={() => setCurrentStep(1)}
          onBack={() => {}}
          onSkip={() => {}}
        />
      ),
    },
    {
      title: 'Security',
      content: (
        <SecurityStep
          onNext={() => setCurrentStep(2)}
          onBack={() => setCurrentStep(0)}
          onSkip={() => setCurrentStep(2)}
        />
      ),
    },
    {
      title: 'Bridge Setup',
      content: (
        <BridgeStep
          onNext={() => setCurrentStep(3)}
          onBack={() => setCurrentStep(1)}
          onSkip={() => setCurrentStep(3)}
        />
      ),
    },
    {
      title: 'Complete',
      content: (
        <FinishStep
          onNext={() => {
            setIsCompleted(true);
            navigate('/dashboard');
          }}
          onBack={() => setCurrentStep(2)}
          onSkip={() => {}}
        />
      ),
    },
  ];
  
  useEffect(() => {
    // Check if this is first run
    const hasCompletedOnboarding = localStorage.getItem('onboardingCompleted') === 'true';
    if (hasCompletedOnboarding) {
      navigate('/dashboard');
    }
  }, [navigate]);
  
  useEffect(() => {
    if (isCompleted) {
      localStorage.setItem('onboardingCompleted', 'true');
    }
  }, [isCompleted]);
  
  return (
    <div className="onboarding-wizard">
      <Progress percent={(currentStep / (steps.length - 1)) * 100} />
      
      <Steps current={currentStep}>
        {steps.map(step => (
          <Steps.Step key={step.title} title={step.title} />
        ))}
      </Steps>
      
      <div className="steps-content">
        {steps[currentStep].content}
      </div>
    </div>
  );
};
