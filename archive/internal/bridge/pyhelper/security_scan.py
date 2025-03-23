#!/usr/bin/env python3

import sys
import json
import socket
import ssl
import requests
import os
import re
import hashlib
import magic
import subprocess
from datetime import datetime
from typing import Dict, List, Any
from dataclasses import dataclass, asdict
from pathlib import Path

@dataclass
class SecurityIssue:
    type: str
    description: str
    severity: str
    file_path: str = ""
    line_number: int = 0
    recommendation: str = ""

@dataclass
class ScanResult:
    timestamp: str
    issues: List[SecurityIssue]
    summary: Dict[str, Any]
    is_secure: bool

class SecurityScanner:
    def __init__(self):
        self.known_vulnerabilities = self._load_vulnerability_database()
        self.malware_signatures = self._load_malware_signatures()
        self.sensitive_patterns = {
            'api_key': r'(?i)(api[_-]key|apikey|secret)["\']?\s*(?::|=)\s*["\']([^"\']+)',
            'password': r'(?i)(password|passwd|pwd)["\']?\s*(?::|=)\s*["\']([^"\']+)',
            'private_key': r'-----BEGIN (?:RSA )?PRIVATE KEY-----',
            'token': r'(?i)(access_token|auth_token|jwt)["\']?\s*(?::|=)\s*["\']([^"\']+)',
        }

    def _load_vulnerability_database(self) -> Dict[str, Any]:
        # TODO: Load from actual vulnerability database
        return {}

    def _load_malware_signatures(self) -> List[str]:
        # TODO: Load from actual signature database
        return []

    def scan_project(self, project_path: str) -> ScanResult:
        issues = []
        
        # Scan project structure
        issues.extend(self._scan_project_structure(project_path))
        
        # Scan dependencies
        issues.extend(self._scan_dependencies(project_path))
        
        # Scan for sensitive information
        issues.extend(self._scan_sensitive_info(project_path))
        
        # Scan for malware
        issues.extend(self._scan_malware(project_path))
        
        # Scan for common vulnerabilities
        issues.extend(self._scan_vulnerabilities(project_path))

        # Generate summary
        summary = self._generate_summary(issues)
        
        return ScanResult(
            timestamp=datetime.utcnow().isoformat(),
            issues=issues,
            summary=summary,
            is_secure=len([i for i in issues if i.severity in ('high', 'critical')]) == 0
        )

    def _scan_project_structure(self, path: str) -> List[SecurityIssue]:
        issues = []
        
        # Check for suspicious file permissions
        for root, dirs, files in os.walk(path):
            for item in dirs + files:
                item_path = os.path.join(root, item)
                try:
                    stats = os.stat(item_path)
                    if stats.st_mode & 0o777 == 0o777:
                        issues.append(SecurityIssue(
                            type="excessive_permissions",
                            description=f"File has excessive permissions: {item_path}",
                            severity="medium",
                            file_path=item_path,
                            recommendation="Restrict file permissions to minimum required"
                        ))
                except OSError:
                    continue

        return issues

    def _scan_dependencies(self, path: str) -> List[SecurityIssue]:
        issues = []
        
        # Check package files
        dependency_files = [
            'requirements.txt',
            'package.json',
            'Gemfile',
            'go.mod',
            'pom.xml'
        ]
        
        for dep_file in dependency_files:
            dep_path = os.path.join(path, dep_file)
            if os.path.exists(dep_path):
                # TODO: Implement actual dependency checking against vulnerability database
                pass

        return issues

    def _scan_sensitive_info(self, path: str) -> List[SecurityIssue]:
        issues = []
        
        for root, _, files in os.walk(path):
            for file in files:
                if file.startswith('.') or file.endswith(('.jpg', '.png', '.gif', '.pdf')):
                    continue
                    
                file_path = os.path.join(root, file)
                try:
                    with open(file_path, 'r', encoding='utf-8') as f:
                        content = f.read()
                        
                        for pattern_name, pattern in self.sensitive_patterns.items():
                            matches = re.finditer(pattern, content)
                            for match in matches:
                                issues.append(SecurityIssue(
                                    type="sensitive_information",
                                    description=f"Found potential {pattern_name} in file",
                                    severity="high",
                                    file_path=file_path,
                                    line_number=content.count('\n', 0, match.start()) + 1,
                                    recommendation=f"Remove {pattern_name} from source code and use environment variables or secure secrets management"
                                ))
                except (UnicodeDecodeError, IOError):
                    continue

        return issues

    def _scan_malware(self, path: str) -> List[SecurityIssue]:
        issues = []
        
        for root, _, files in os.walk(path):
            for file in files:
                file_path = os.path.join(root, file)
                try:
                    # Check file type
                    file_type = magic.from_file(file_path)
                    
                    # Skip known safe files
                    if any(safe in file_type.lower() for safe in ['text', 'image', 'empty']):
                        continue
                    
                    # Calculate file hash
                    with open(file_path, 'rb') as f:
                        file_hash = hashlib.sha256(f.read()).hexdigest()
                        
                        # Check against malware signatures
                        if file_hash in self.malware_signatures:
                            issues.append(SecurityIssue(
                                type="malware_detected",
                                description=f"Potential malware detected: {file_path}",
                                severity="critical",
                                file_path=file_path,
                                recommendation="Remove malicious file immediately"
                            ))
                except (IOError, magic.MagicException):
                    continue

        return issues

    def _scan_vulnerabilities(self, path: str) -> List[SecurityIssue]:
        issues = []
        
        # Scan for common web vulnerabilities in code
        vulnerability_patterns = {
            'sql_injection': r'(?i)(?:execute|exec)\s*\(\s*["\']?\s*SELECT',
            'xss': r'(?i)innerHTML\s*=|document\.write\s*\(',
            'command_injection': r'(?i)(?:system|exec|eval)\s*\(',
            'path_traversal': r'\.\./',
        }
        
        for root, _, files in os.walk(path):
            for file in files:
                if not file.endswith(('.js', '.py', '.php', '.rb', '.go')):
                    continue
                    
                file_path = os.path.join(root, file)
                try:
                    with open(file_path, 'r', encoding='utf-8') as f:
                        content = f.read()
                        
                        for vuln_type, pattern in vulnerability_patterns.items():
                            matches = re.finditer(pattern, content)
                            for match in matches:
                                issues.append(SecurityIssue(
                                    type=f"potential_{vuln_type}",
                                    description=f"Potential {vuln_type} vulnerability detected",
                                    severity="high",
                                    file_path=file_path,
                                    line_number=content.count('\n', 0, match.start()) + 1,
                                    recommendation=f"Review and fix potential {vuln_type} vulnerability"
                                ))
                except (UnicodeDecodeError, IOError):
                    continue

        return issues

    def _generate_summary(self, issues: List[SecurityIssue]) -> Dict[str, Any]:
        severity_counts = {'critical': 0, 'high': 0, 'medium': 0, 'low': 0}
        issue_types = {}
        
        for issue in issues:
            severity_counts[issue.severity] += 1
            issue_types[issue.type] = issue_types.get(issue.type, 0) + 1

        return {
            'total_issues': len(issues),
            'severity_counts': severity_counts,
            'issue_types': issue_types,
            'recommendations': len([i for i in issues if i.recommendation])
        }

    def check_ssl(self, domain: str) -> None:
        try:
            context = ssl.create_default_context()
            with socket.create_connection((domain, 443)) as sock:
                with context.wrap_socket(sock, server_hostname=domain) as ssock:
                    cert = ssock.getpeercert()
                    return {
                        'version': ssock.version(),
                        'cipher': ssock.cipher(),
                        'expiry': cert['notAfter'],
                        'issuer': dict(x[0] for x in cert['issuer'])
                    }
        except Exception as e:
            return {'error': str(e)}

    def check_security_headers(self, url: str) -> Dict[str, Any]:
        try:
            response = requests.get(url)
            headers = response.headers
            
            security_headers = {
                'Strict-Transport-Security': False,
                'Content-Security-Policy': False,
                'X-Frame-Options': False,
                'X-Content-Type-Options': False,
                'X-XSS-Protection': False,
                'Referrer-Policy': False,
                'Permissions-Policy': False
            }

            for header in security_headers:
                security_headers[header] = header in headers

            return security_headers
        except Exception as e:
            return {'error': str(e)}

def main():
    if len(sys.argv) < 3:
        print(json.dumps({'error': 'Invalid arguments'}))
        sys.exit(1)

    command = sys.argv[1]
    target = sys.argv[2]

    scanner = SecurityScanner()

    if command == 'scan-project':
        result = scanner.scan_project(target)
        print(json.dumps(asdict(result)))
    elif command == 'ssl-check':
        result = scanner.check_ssl(target)
        print(json.dumps(result))
    elif command == 'headers-check':
        result = scanner.check_security_headers(target)
        print(json.dumps(result))
    else:
        print(json.dumps({'error': 'Invalid command'}))
        sys.exit(1)

if __name__ == '__main__':
    main()
