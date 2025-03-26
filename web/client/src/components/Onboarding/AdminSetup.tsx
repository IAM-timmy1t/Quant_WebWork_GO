/**
 * AdminSetup.tsx
 * 
 * @module components/Onboarding
 * @description Component for configuring admin credentials and settings during onboarding.
 * @version 1.0.0
 */

import React, { useState } from 'react';
import { Form, Input, Button, Card, Alert, Checkbox, Space, Divider } from '../ui';
import { EyeOutlined, EyeInvisibleOutlined, LockOutlined, UserOutlined } from '@ant-design/icons';
import { useConfig } from '../../hooks/useConfig';
import { useAuth } from '../../hooks/useAuth';

interface AdminSetupProps {
  onComplete?: () => void;
}

export const AdminSetup: React.FC<AdminSetupProps> = ({ onComplete }) => {
  const { createAdmin } = useAuth();
  const { config, updateConfig } = useConfig();
  
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [email, setEmail] = useState('');
  const [rememberDevice, setRememberDevice] = useState(false);
  const [mfaEnabled, setMfaEnabled] = useState(false);
  const [loading, setLoading] = useState(false);
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [confirmVisible, setConfirmVisible] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [passwordStrength, setPasswordStrength] = useState({
    score: 0,
    feedback: 'Password strength is too weak',
    color: 'red'
  });
  
  // Password strength checker
  const checkPasswordStrength = (password: string) => {
    // This is a simple example. In a real app, you would use a library like zxcvbn
    let score = 0;
    let feedback = 'Password is too weak';
    let color = 'red';
    
    if (password.length >= 8) score += 1;
    if (password.match(/[A-Z]/)) score += 1;
    if (password.match(/[a-z]/)) score += 1;
    if (password.match(/[0-9]/)) score += 1;
    if (password.match(/[^A-Za-z0-9]/)) score += 1;
    
    if (score === 5) {
      feedback = 'Password strength is excellent';
      color = 'green';
    } else if (score >= 3) {
      feedback = 'Password strength is good';
      color = 'orange';
    } else {
      feedback = 'Password strength is too weak';
      color = 'red';
    }
    
    setPasswordStrength({ score, feedback, color });
    return score;
  };
  
  // Handle password change
  const handlePasswordChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newPassword = e.target.value;
    setPassword(newPassword);
    checkPasswordStrength(newPassword);
  };
  
  // Handle form submission
  const handleSubmit = async () => {
    setError(null);
    
    // Validate inputs
    if (!username || !password || !confirmPassword || !email) {
      setError('All fields are required');
      return;
    }
    
    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }
    
    if (passwordStrength.score < 3) {
      setError('Please create a stronger password');
      return;
    }
    
    setLoading(true);
    
    try {
      // In a real application, this would call your API
      const created = await createAdmin({
        username,
        password,
        email,
        mfaEnabled,
        rememberDevice
      });
      
      if (created) {
        setSuccess(true);
        if (onComplete) {
          onComplete();
        }
        
        // Update security config to indicate admin is set up
        updateConfig('security.adminConfigured', true);
      } else {
        setError('Failed to create admin account. Please try again.');
      }
    } catch (err: any) {
      setError(err.message || 'An unexpected error occurred');
    } finally {
      setLoading(false);
    }
  };
  
  return (
    <Card className="admin-setup" title="Administrator Account Setup">
      {success ? (
        <Alert
          type="success"
          message="Admin Account Created Successfully"
          description="Your administrator account has been set up. You can now use these credentials to access the admin dashboard."
          showIcon
        />
      ) : (
        <>
          {error && (
            <Alert 
              type="error" 
              message={error} 
              showIcon 
              style={{ marginBottom: 16 }} 
            />
          )}
          
          <Form layout="vertical">
            <Form.Item 
              label="Username" 
              required 
              tooltip="Choose a unique admin username"
            >
              <Input
                prefix={<UserOutlined />}
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="admin"
              />
            </Form.Item>
            
            <Form.Item 
              label="Email Address" 
              required 
              tooltip="Used for account recovery and security notifications"
            >
              <Input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="admin@example.com"
              />
            </Form.Item>
            
            <Form.Item 
              label="Password" 
              required 
              tooltip="Create a strong password with at least 8 characters"
            >
              <Input.Password
                prefix={<LockOutlined />}
                value={password}
                onChange={handlePasswordChange}
                placeholder="Strong password"
                visibilityToggle={{
                  visible: passwordVisible,
                  onVisibleChange: setPasswordVisible,
                }}
                iconRender={(visible) => (visible ? <EyeOutlined /> : <EyeInvisibleOutlined />)}
              />
              {password && (
                <div className="password-strength">
                  <div 
                    className="strength-bar" 
                    style={{ 
                      width: `${(passwordStrength.score / 5) * 100}%`,
                      backgroundColor: passwordStrength.color 
                    }} 
                  />
                  <span style={{ color: passwordStrength.color }}>
                    {passwordStrength.feedback}
                  </span>
                </div>
              )}
            </Form.Item>
            
            <Form.Item 
              label="Confirm Password" 
              required
            >
              <Input.Password
                prefix={<LockOutlined />}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Confirm password"
                visibilityToggle={{
                  visible: confirmVisible,
                  onVisibleChange: setConfirmVisible,
                }}
                iconRender={(visible) => (visible ? <EyeOutlined /> : <EyeInvisibleOutlined />)}
              />
              {confirmPassword && password !== confirmPassword && (
                <span style={{ color: 'red' }}>Passwords do not match</span>
              )}
            </Form.Item>
            
            <Divider>Security Options</Divider>
            
            <Form.Item>
              <Checkbox
                checked={mfaEnabled}
                onChange={(e) => setMfaEnabled(e.target.checked)}
              >
                Enable Multi-Factor Authentication (Recommended)
              </Checkbox>
            </Form.Item>
            
            <Form.Item>
              <Checkbox
                checked={rememberDevice}
                onChange={(e) => setRememberDevice(e.target.checked)}
              >
                Remember this device for 30 days
              </Checkbox>
            </Form.Item>
            
            <Form.Item>
              <Space>
                <Button 
                  type="primary" 
                  onClick={handleSubmit} 
                  loading={loading}
                >
                  Create Admin Account
                </Button>
              </Space>
            </Form.Item>
          </Form>
          
          <div className="admin-setup-notes">
            <Alert
              type="info"
              message="Why is this important?"
              description="The administrator account allows you to manage users, configure system settings, and monitor security. Make sure to use a strong password and keep your credentials secure."
              showIcon
            />
          </div>
        </>
      )}
    </Card>
  );
};
