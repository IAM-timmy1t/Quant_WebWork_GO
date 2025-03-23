"""
Enhanced Project Tree Generator v3.1
-----------------------------------
Architectural analysis and feature mapping system with token-optimized output
and comprehensive build plan integration.

Key Capabilities:
1. Project Structure Analysis
   - Hierarchical tree generation
   - Module dependency tracking
   - Token-conscious metadata collection

2. Feature Management
   - Core/optional feature categorization
   - Complexity analysis
   - Token impact assessment

3. Build Plan Integration
   - Configuration alignment
   - Feature priority mapping
   - Resource optimization

Dependencies:
    rich>=10.0.0   : Console interface and progress tracking
    Python>=3.9    : Type hinting and async features

Author: QUANT_AGI Development Team
Last Updated: 2025-02-17
"""

import os
import sys
import json
import ast
import logging
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Set, Any, Tuple, Union
from dataclasses import dataclass, field, asdict
from collections import defaultdict

# Rich UI components
from rich import print as rprint
from rich.console import Console
from rich.tree import Tree
from rich.panel import Panel
from rich.table import Table
from rich.prompt import Prompt, Confirm
from rich.progress import Progress, SpinnerColumn, TextColumn, BarColumn
from rich.logging import RichHandler
from rich.style import Style

# Configure logging with rich handler
logging.basicConfig(
    level=logging.INFO,
    format="%(message)s",
    handlers=[RichHandler(rich_tracebacks=True)]
)
logger = logging.getLogger("project_tree")

@dataclass
class FeatureCategory:
    """Feature category definition with validation rules."""
    name: str
    description: str
    priority: int
    validation_rules: Dict[str, Any]
    token_allocation: Dict[str, int]
    required_elements: Set[str] = field(default_factory=set)

    def validate_feature(self, feature_data: Dict[str, Any]) -> Tuple[bool, str]:
        """Validate feature against category rules."""
        missing_elements = self.required_elements - set(feature_data.keys())
        if missing_elements:
            return False, f"Missing required elements: {missing_elements}"
            
        if feature_data.get('token_impact', 0) > self.token_allocation.get('max', float('inf')):
            return False, "Exceeds token allocation"
            
        return True, "Valid feature"

@dataclass
class AnalysisConfiguration:
    """Comprehensive analysis configuration."""
    view_mode: str = "standard"  # standard, high-level, focused
    max_depth: int = -1
    focus_path: Optional[str] = None
    analyze_features: bool = True
    show_metrics: bool = True
    token_budget: int = 8000
    
    # Feature analysis settings
    feature_categories: Dict[str, FeatureCategory] = field(default_factory=lambda: {
        "core": FeatureCategory(
            name="Core Features",
            description="Essential system functionalities",
            priority=1,
            validation_rules={"required_docs": True, "test_coverage": 0.9},
            token_allocation={"max": 3000, "per_feature": 500},
            required_elements={"description", "dependencies"}
        ),
        "supporting": FeatureCategory(
            name="Supporting Features",
            description="Enhancement functionalities",
            priority=2,
            validation_rules={"required_docs": True, "test_coverage": 0.7},
            token_allocation={"max": 2000, "per_feature": 300},
            required_elements={"description"}
        ),
        "utility": FeatureCategory(
            name="Utility Features",
            description="Helper functionalities",
            priority=3,
            validation_rules={"required_docs": False, "test_coverage": 0.5},
            token_allocation={"max": 1000, "per_feature": 200},
            required_elements=set()
        )
    })
    
    exclude_patterns: Set[str] = field(default_factory=lambda: {
        "__pycache__", ".git", "venv", "build", "dist"
    })

class FeatureAnalyzer:
    """Feature analysis and validation system."""
    
    def __init__(self, config: AnalysisConfiguration):
        self.config = config
        self.features: Dict[str, Dict[str, Any]] = defaultdict(dict)
        self.validation_results: Dict[str, List[str]] = defaultdict(list)

    def analyze_node(
        self,
        node: Union[ast.ClassDef, ast.FunctionDef],
        module_path: str
    ) -> Optional[Dict[str, Any]]:
        """Analyze AST node for feature metadata."""
        try:
            docstring = ast.get_docstring(node)
            if not docstring:
                return None

            # Extract basic metadata
            feature_data = {
                "name": node.name,
                "description": docstring.split("\n")[0],
                "module_path": module_path,
                "type": self._determine_feature_type(node),
                "complexity": self._calculate_complexity(node),
                "dependencies": self._extract_dependencies(node),
                "token_impact": self._estimate_token_impact(node)
            }

            # Validate against category rules
            category = self.config.feature_categories[feature_data["type"]]
            is_valid, message = category.validate_feature(feature_data)
            
            if not is_valid:
                self.validation_results[module_path].append(
                    f"Feature '{node.name}': {message}"
                )
            
            feature_data["validation_status"] = is_valid
            self.features[module_path][node.name] = feature_data
            
            return feature_data
            
        except Exception as e:
            logger.warning(f"Error analyzing node {getattr(node, 'name', 'unknown')}: {e}")
            return None

    def _determine_feature_type(self, node: ast.AST) -> str:
        """Determine feature type from node characteristics."""
        if isinstance(node, ast.ClassDef):
            return "core"
        elif self._is_utility_function(node):
            return "utility"
        else:
            return "supporting"

    def _is_utility_function(self, node: ast.AST) -> bool:
        """Check if node represents a utility function."""
        if not isinstance(node, ast.FunctionDef):
            return False
            
        # Check for utility indicators
        utility_prefixes = {"_", "util_", "helper_"}
        return any(node.name.startswith(prefix) for prefix in utility_prefixes)

    def _calculate_complexity(self, node: ast.AST) -> int:
        """Calculate node complexity score."""
        complexity = 1
        for child in ast.walk(node):
            if isinstance(child, (ast.If, ast.While, ast.For, ast.Try)):
                complexity += 1
            elif isinstance(child, ast.FunctionDef):
                complexity += 1
        return min(complexity, 5)

    def _extract_dependencies(self, node: ast.AST) -> List[str]:
        """Extract node dependencies."""
        dependencies = []
        for child in ast.walk(node):
            if isinstance(child, ast.Import):
                for name in child.names:
                    dependencies.append(name.name)
            elif isinstance(child, ast.ImportFrom):
                if child.module:
                    dependencies.append(child.module)
        return list(set(dependencies))

    def _estimate_token_impact(self, node: ast.AST) -> int:
        """Estimate token impact of node."""
        source_lines = len(ast.unparse(node).split("\n"))
        docstring_lines = len(ast.get_docstring(node).split("\n")) if ast.get_docstring(node) else 0
        
        # Estimate tokens based on code structure
        base_tokens = source_lines * 10  # Average tokens per line
        doc_tokens = docstring_lines * 8  # Average tokens per docstring line
        
        return base_tokens + doc_tokens

    def get_feature_summary(self) -> Dict[str, Any]:
        """Generate comprehensive feature summary."""
        summary = {
            "categories": {},
            "validation": {
                "total_features": 0,
                "valid_features": 0,
                "validation_messages": self.validation_results
            },
            "token_analysis": {
                "total_impact": 0,
                "by_category": {}
            }
        }
        
        # Aggregate features by category
        for module_features in self.features.values():
            for feature in module_features.values():
                cat_name = feature["type"]
                if cat_name not in summary["categories"]:
                    summary["categories"][cat_name] = []
                    
                summary["categories"][cat_name].append({
                    "name": feature["name"],
                    "description": feature["description"],
                    "complexity": feature["complexity"],
                    "token_impact": feature["token_impact"],
                    "validation_status": feature["validation_status"]
                })
                
                # Update statistics
                summary["validation"]["total_features"] += 1
                if feature["validation_status"]:
                    summary["validation"]["valid_features"] += 1
                    
                summary["token_analysis"]["total_impact"] += feature["token_impact"]
                summary["token_analysis"]["by_category"][cat_name] = (
                    summary["token_analysis"]["by_category"].get(cat_name, 0) +
                    feature["token_impact"]
                )
        
        return summary

class ProjectAnalyzer:
    """Project structure and feature analysis system."""
    
    def __init__(
        self,
        project_root: Path,
        config: Optional[AnalysisConfiguration] = None
    ):
        """Initialize project analyzer.
        
        Args:
            project_root: Root directory of project
            config: Optional analysis configuration
        """
        self.project_root = Path(project_root)
        self.config = config or AnalysisConfiguration()
        self.console = Console()
        
        # Analysis components
        self.feature_analyzer = FeatureAnalyzer(self.config)
        
        # Progress tracking
        self.progress: Optional[Progress] = None
        self.current_task_id: Optional[str] = None

    def configure_interactively(self) -> None:
        """Configure analysis options through interactive prompts."""
        self.console.print("\n[bold cyan]Analysis Configuration[/bold cyan]")
        
        # Configure analysis depth
        depth_str = Prompt.ask(
            "Maximum depth to analyze (-1 for unlimited)",
            default=str(self.config.max_depth)
        )
        try:
            self.config.max_depth = int(depth_str)
        except ValueError:
            self.console.print("[yellow]Invalid depth, using default (-1)[/yellow]")
        
        # Configure analysis detail
        self.config.show_metrics = Confirm.ask(
            "Show detailed metrics?",
            default=True
        )
        
        # Configure token budget
        if self.config.show_metrics:
            budget_str = Prompt.ask(
                "Token budget per module",
                default=str(self.config.token_budget)
            )
            try:
                self.config.token_budget = int(budget_str)
            except ValueError:
                self.console.print("[yellow]Invalid budget, using default[/yellow]")
        
        # Configure view mode
        self.config.view_mode = Prompt.ask(
            "Select view mode",
            choices=["standard", "high-level", "focused"],
            default=self.config.view_mode
        )
        
        if self.config.view_mode == "focused":
            self.config.focus_path = Prompt.ask(
                "Enter module path to focus on",
                default=str(self.config.focus_path or "")
            )

    def analyze_project(self) -> Tuple[Tree, Dict[str, Any]]:
        """Analyze project structure and features."""
        try:
            with Progress(
                SpinnerColumn(),
                TextColumn("[progress.description]{task.description}"),
                BarColumn(),
                TextColumn("[progress.percentage]{task.percentage:>3.0f}%"),
                console=self.console
            ) as progress:
                self.progress = progress
                self.current_task_id = progress.add_task(
                    "Analyzing project structure...",
                    total=100
                )
                
                # Validate and prepare root path
                root_path = (
                    self.project_root / self.config.focus_path
                    if self.config.focus_path
                    else self.project_root
                )
                
                if not root_path.exists():
                    raise ValueError(f"Path does not exist: {root_path}")
                
                # Generate tree structure
                progress.update(self.current_task_id, advance=20)
                tree = self._generate_tree(root_path)
                
                # Analyze features
                progress.update(self.current_task_id, advance=40)
                feature_summary = self.feature_analyzer.get_feature_summary()
                
                # Finalize analysis
                progress.update(self.current_task_id, advance=40)
                
                return tree, feature_summary
                
        except Exception as e:
            logger.error(f"Error analyzing project: {e}")
            raise
        finally:
            self.progress = None
            self.current_task_id = None

    def _generate_tree(self, root_path: Path) -> Tree:
        """Generate Rich tree structure."""
        tree = Tree(
            f"[bold blue]{root_path.name}[/]",
            guide_style="bold bright_blue"
        )
        
        def add_to_tree(path: Path, tree: Tree, depth: int = 0) -> None:
            if self.config.max_depth != -1 and depth > self.config.max_depth:
                return
                
            try:
                for item in sorted(path.iterdir()):
                    if item.name in self.config.exclude_patterns:
                        continue
                        
                    # Create node label
                    if item.is_dir():
                        label = f"[bold blue]{item.name}[/]"
                        if self.config.show_metrics:
                            features = self.feature_analyzer.features.get(str(item), {})
                            if features:
                                label += (f" [dim](Features: {len(features)}, "
                                        f"Valid: {sum(1 for f in features.values() if f['validation_status'])})[/]")
                    else:
                        label = item.name
                        
                    # Add node and recurse
                    branch = tree.add(label)
                    if item.is_dir():
                        add_to_tree(item, branch, depth + 1)
                        
            except PermissionError:
                tree.add("[red][Access Denied][/]")
            except Exception as e:
                logger.warning(f"Error processing {path}: {e}")
                
        add_to_tree(root_path, tree)
        return tree

    def _analyze_file(self, file_path: Path) -> None:
        """Analyze Python file for features.
        
        Args:
            file_path: Path to the Python file to analyze
            
        This method parses the Python file and analyzes its AST for classes
        and functions, extracting feature metadata using the feature analyzer.
        """
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                tree = ast.parse(f.read())
                
            for node in ast.walk(tree):
                if isinstance(node, (ast.ClassDef, ast.FunctionDef)):
                    self.feature_analyzer.analyze_node(
                        node,
                        str(file_path.relative_to(self.project_root))
                    )
        except Exception as e:
            logger.error(f"Error analyzing file {file_path}: {e}")

    def display_results(self, tree: Tree, feature_summary: Dict[str, Any]) -> None:
        """Display comprehensive analysis results with structured formatting.
        
        Feature Display Strategy:
        1. Project Structure Overview
        2. Feature Category Analysis
        3. Validation Statistics
        4. Token Impact Assessment
        
        Args:
            tree: Generated project structure tree
            feature_summary: Comprehensive feature analysis results
        """
        self.console.print("\n[bold cyan]Project Structure Analysis[/bold cyan]")
        self.console.print(tree)
        
        if self.config.show_metrics:
            self._display_feature_analysis(feature_summary)
            
            if Confirm.ask("\nShow detailed validation results?"):
                self._display_validation_results(feature_summary)

    def _display_feature_analysis(self, feature_summary: Dict[str, Any]) -> None:
        """Display structured feature analysis with category breakdowns.
        
        Layout:
        1. Category Overview Table
        2. Token Impact Analysis
        3. Implementation Progress
        """
        self.console.print("\n[bold cyan]Feature Analysis Summary[/bold cyan]")
        
        # Create category overview table
        table = Table(show_header=True, header_style="bold magenta")
        table.add_column("Category", style="cyan")
        table.add_column("Features", justify="right", style="green")
        table.add_column("Valid", justify="right", style="blue")
        table.add_column("Token Impact", justify="right", style="yellow")
        
        # Add category rows
        total_features = 0
        total_valid = 0
        total_tokens = 0
        
        for category, features in feature_summary["categories"].items():
            valid_count = sum(1 for f in features if f["validation_status"])
            token_impact = feature_summary["token_analysis"]["by_category"].get(category, 0)
            
            table.add_row(
                self.config.feature_categories[category].name,
                str(len(features)),
                f"{valid_count} ({valid_count/len(features):.0%})",
                str(token_impact)
            )
            
            total_features += len(features)
            total_valid += valid_count
            total_tokens += token_impact
        
        # Add summary row
        table.add_row(
            "Total",
            str(total_features),
            f"{total_valid} ({total_valid/total_features:.0%})",
            str(total_tokens),
            style="bold"
        )
        
        self.console.print(table)

    def _display_validation_results(self, feature_summary: Dict[str, Any]) -> None:
        """Display detailed validation results with issue tracking.
        
        Validation Display:
        1. Module-level Issues
        2. Feature-specific Validation
        3. Token Budget Analysis
        """
        self.console.print("\n[bold cyan]Validation Details[/bold cyan]")
        
        validation_messages = feature_summary["validation"]["validation_messages"]
        if not validation_messages:
            self.console.print("[green]All features passed validation![/green]")
            return
        
        # Display validation issues by module
        for module_path, messages in validation_messages.items():
            if messages:
                self.console.print(f"\n[yellow]Module: {module_path}[/yellow]")
                for msg in messages:
                    self.console.print(f"  • {msg}")

    def export_results(self, output_path: Path) -> None:
        """Export analysis results with comprehensive metadata.
        
        Export Structure:
        1. Project Metadata
        2. Feature Analysis
        3. Validation Results
        4. Token Usage Statistics
        
        Args:
            output_path: Target path for results export
        """
        try:
            # Generate export data
            export_data = {
                "metadata": {
                    "timestamp": datetime.now().isoformat(),
                    "project_root": str(self.project_root),
                    "configuration": asdict(self.config)
                },
                "feature_analysis": {
                    "categories": {
                        category.name: {
                            "description": category.description,
                            "priority": category.priority,
                            "token_allocation": category.token_allocation,
                            "validation_rules": category.validation_rules
                        }
                        for category in self.config.feature_categories.values()
                    },
                    "features": self.feature_analyzer.features,
                    "validation_results": self.feature_analyzer.validation_results
                }
            }
            
            # Ensure directory exists
            output_path.parent.mkdir(parents=True, exist_ok=True)
            
            # Write results
            with open(output_path, 'w') as f:
                json.dump(export_data, f, indent=2)
                
            self.console.print(f"\n[green]Analysis results exported to {output_path}[/green]")
            
        except Exception as e:
            logger.error(f"Error exporting results: {e}")
            self.console.print("[red]Error exporting results. Check logs for details.[/red]")

def main():
    """Main execution with structured analysis workflow.
    
    Analysis Process:
    1. Project Configuration
    2. Feature Analysis
    3. Results Visualization
    4. Optional Export
    """
    console = Console()
    
    try:
        # Project root configuration
        project_root = Prompt.ask(
            "Enter project root path",
            default=str(Path.cwd())
        )
        
        # Validate project path
        if not Path(project_root).exists():
            console.print("[red]Error: Project root does not exist[/red]")
            return
        
        # Initialize analyzer
        try:
            analyzer = ProjectAnalyzer(Path(project_root))
        except Exception as e:
            console.print(f"[red]Error initializing analyzer: {str(e)}[/red]")
            return
        
        # Configure analysis
        analyzer.configure_interactively()
        
        # Perform analysis with progress tracking
        try:
            tree, feature_summary = analyzer.analyze_project()
            analyzer.display_results(tree, feature_summary)
            
            # Export results if requested
            if Confirm.ask("Export analysis results?"):
                timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
                default_export = Path(project_root) / f"project_analysis_{timestamp}.json"
                
                export_path = Prompt.ask(
                    "Export path",
                    default=str(default_export)
                )
                analyzer.export_results(Path(export_path))
                
        except ValueError as e:
            console.print(f"[red]Analysis error: {e}[/red]")
        except Exception as e:
            logger.exception("Unexpected error during analysis")
            console.print("[red]An unexpected error occurred. Check logs for details.[/red]")
            
    except KeyboardInterrupt:
        console.print("\n[yellow]Analysis cancelled by user[/yellow]")
    except Exception as e:
        logger.exception("Critical error")
        console.print("[red]A critical error occurred. Check logs for details.[/red]")

if __name__ == "__main__":
    main()