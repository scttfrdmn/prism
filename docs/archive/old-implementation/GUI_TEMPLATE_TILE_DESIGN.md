# Prism Template Tile System Design

## Version: v0.4.3+ Enhancement  
**Feature**: Template Complexity Levels & Visual Tile Interface  
**Target**: Enhanced Progressive Disclosure UX for Academic Researchers

---

## 🎯 **Design Goals**

### **Primary Objectives**
- **Intuitive Selection**: Visual tiles that clearly communicate template purpose and complexity
- **Progressive Disclosure**: Beginner templates prominently featured, advanced options available but not overwhelming
- **Academic Workflow**: Templates organized by research domains and skill progression
- **Visual Clarity**: Consistent iconography, color coding, and complexity indicators

### **User Experience Principles**
- **Beginner-Friendly**: New users see simple, well-tested templates first
- **Expert-Accessible**: Power users can easily find advanced configurations
- **Domain-Aware**: Templates grouped by research discipline (ML, R, Genomics, etc.)
- **Progressive Learning**: Clear path from basic to advanced templates

---

## 📊 **Template Complexity System**

### **Complexity Levels**

```javascript
const COMPLEXITY_LEVELS = {
  SIMPLE: {
    level: 1,
    label: "Simple",
    description: "Perfect for getting started - pre-configured and tested",
    icon: "🟢",
    color: "#059669",
    badge: "Ready to Use"
  },
  
  MODERATE: {
    level: 2, 
    label: "Moderate",
    description: "Some customization available - good for regular users",
    icon: "🟡", 
    color: "#d97706",
    badge: "Some Options"
  },
  
  ADVANCED: {
    level: 3,
    label: "Advanced", 
    description: "Highly configurable - for experienced users",
    icon: "🟠",
    color: "#ea580c", 
    badge: "Many Options"
  },
  
  COMPLEX: {
    level: 4,
    label: "Complex",
    description: "Maximum flexibility - requires technical knowledge", 
    icon: "🔴",
    color: "#dc2626",
    badge: "Full Control"
  }
}
```

### **Template Categories with Complexity Distribution**

```javascript
const TEMPLATE_CATEGORIES = {
  "Machine Learning": {
    icon: "🤖",
    color: "#2563eb",
    templates: [
      { name: "Python ML Quickstart", complexity: "SIMPLE", popular: true },
      { name: "PyTorch Research Environment", complexity: "MODERATE" },
      { name: "Custom CUDA Toolkit", complexity: "ADVANCED" },
      { name: "Multi-GPU Distributed Training", complexity: "COMPLEX" }
    ]
  },
  
  "Data Science": {
    icon: "📊", 
    color: "#7c3aed",
    templates: [
      { name: "R Statistical Analysis", complexity: "SIMPLE", popular: true },
      { name: "Jupyter Data Science Stack", complexity: "SIMPLE", popular: true },
      { name: "Apache Spark Analytics", complexity: "MODERATE" },
      { name: "Custom R Packages & Compilation", complexity: "ADVANCED" }
    ]
  },
  
  "Bioinformatics": {
    icon: "🧬",
    color: "#059669", 
    templates: [
      { name: "Genomics Analysis Pipeline", complexity: "MODERATE" },
      { name: "Proteomics Workflow", complexity: "ADVANCED" },
      { name: "Custom Bioconductor Environment", complexity: "COMPLEX" }
    ]
  },
  
  "Web Development": {
    icon: "🌐",
    color: "#0891b2",
    templates: [
      { name: "Node.js Development", complexity: "SIMPLE" },
      { name: "Full-Stack Research Portal", complexity: "MODERATE" },
      { name: "Microservices Architecture", complexity: "ADVANCED" }
    ]
  },
  
  "Base Systems": {
    icon: "🖥️", 
    color: "#64748b",
    templates: [
      { name: "Ubuntu Desktop", complexity: "SIMPLE" },
      { name: "Rocky Linux Workstation", complexity: "MODERATE" }, 
      { name: "Custom Kernel & Drivers", complexity: "COMPLEX" }
    ]
  }
}
```

---

## 🎨 **Visual Tile Design**

### **Tile Layout Structure**

```html
<div class="template-tile" data-complexity="simple" data-category="ml">
  <!-- Complexity Badge (Top Right) -->
  <div class="complexity-badge simple">
    <span class="complexity-icon">🟢</span>
    <span class="complexity-label">Ready to Use</span>
  </div>
  
  <!-- Popular Badge (Top Left, if applicable) -->
  <div class="popular-badge">⭐ Popular</div>
  
  <!-- Main Content Area -->
  <div class="tile-header">
    <div class="category-icon">🤖</div>
    <div class="tile-title">Python ML Quickstart</div>
    <div class="category-label">Machine Learning</div>
  </div>
  
  <div class="tile-description">
    Pre-configured Jupyter + scikit-learn + pandas. Perfect for ML simples.
  </div>
  
  <!-- Features List -->
  <div class="tile-features">
    <span class="feature-tag">Jupyter</span>
    <span class="feature-tag">GPU Ready</span>
    <span class="feature-tag">Pre-tested</span>
  </div>
  
  <!-- Footer with Metadata -->
  <div class="tile-footer">
    <div class="launch-time">⚡ ~3 min launch</div>
    <div class="cost-estimate">💰 $0.08/hour</div>
  </div>
  
  <!-- Selection State Overlay -->
  <div class="tile-selection-overlay">
    <div class="selection-checkmark">✓</div>
  </div>
</div>
```

### **CSS Styling System**

```css
/* Base tile styling */
.template-tile {
  position: relative;
  background: var(--color-surface-elevated);
  border: 2px solid var(--color-border);
  border-radius: var(--border-radius-xl);
  padding: var(--spacing-lg);
  cursor: pointer;
  transition: all var(--transition-normal);
  min-height: 280px;
  display: flex;
  flex-direction: column;
}

/* Hover and selection states */
.template-tile:hover {
  border-color: var(--color-primary-light);
  box-shadow: var(--shadow-lg);
  transform: translateY(-4px);
}

.template-tile.selected {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px var(--color-primary-light), var(--shadow-lg);
}

/* Difficulty badge styling */
.complexity-badge {
  position: absolute;
  top: var(--spacing-sm);
  right: var(--spacing-sm);
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-xs) var(--spacing-sm);
  border-radius: var(--border-radius);
  font-size: var(--font-size-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.complexity-badge.simple {
  background: rgba(5, 150, 105, 0.1);
  color: #059669;
  border: 1px solid rgba(5, 150, 105, 0.3);
}

.complexity-badge.moderate {
  background: rgba(217, 119, 6, 0.1); 
  color: #d97706;
  border: 1px solid rgba(217, 119, 6, 0.3);
}

.complexity-badge.advanced {
  background: rgba(234, 88, 12, 0.1);
  color: #ea580c; 
  border: 1px solid rgba(234, 88, 12, 0.3);
}

.complexity-badge.complex {
  background: rgba(220, 38, 38, 0.1);
  color: #dc2626;
  border: 1px solid rgba(220, 38, 38, 0.3);
}

/* Popular badge */
.popular-badge {
  position: absolute;
  top: var(--spacing-sm);
  left: var(--spacing-sm);
  background: linear-gradient(135deg, #fbbf24, #f59e0b);
  color: white;
  padding: var(--spacing-xs) var(--spacing-sm);
  border-radius: var(--border-radius);
  font-size: var(--font-size-xs);
  font-weight: 600;
  box-shadow: var(--shadow-sm);
}

/* Grid layout for tiles */
.template-tiles-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: var(--spacing-lg);
  margin-top: var(--spacing-xl);
}
```

### **Responsive Grid Behavior**

- **Desktop (1200px+)**: 3-4 tiles per row
- **Tablet (768px-1199px)**: 2 tiles per row  
- **Mobile (< 768px)**: 1 tile per row, full-width

---

## 🔍 **Filtering and Organization System**

### **Filter Controls**

```html
<div class="template-filters">
  <!-- Difficulty Filter -->
  <div class="filter-group">
    <label class="filter-label">Difficulty Level</label>
    <div class="filter-buttons">
      <button class="filter-btn active" data-complexity="all">All</button>
      <button class="filter-btn" data-complexity="simple">🟢 Beginner</button>
      <button class="filter-btn" data-complexity="moderate">🟡 Intermediate</button>
      <button class="filter-btn" data-complexity="advanced">🟠 Advanced</button>
      <button class="filter-btn" data-complexity="complex">🔴 Expert</button>
    </div>
  </div>
  
  <!-- Category Filter -->
  <div class="filter-group">
    <label class="filter-label">Research Domain</label>
    <div class="filter-buttons">
      <button class="filter-btn active" data-category="all">All Domains</button>
      <button class="filter-btn" data-category="ml">🤖 Machine Learning</button>
      <button class="filter-btn" data-category="datascience">📊 Data Science</button>
      <button class="filter-btn" data-category="bio">🧬 Bioinformatics</button>
      <button class="filter-btn" data-category="web">🌐 Web Development</button>
      <button class="filter-btn" data-category="base">🖥️ Base Systems</button>
    </div>
  </div>
  
  <!-- Sort Options -->
  <div class="filter-group">
    <label class="filter-label">Sort By</label>
    <select class="sort-select">
      <option value="popularity">Most Popular</option>
      <option value="complexity">Difficulty (Easy → Hard)</option>
      <option value="category">Research Domain</option>
      <option value="cost">Cost (Low → High)</option>
      <option value="launch-time">Launch Time (Fast → Slow)</option>
    </select>
  </div>
</div>
```

### **Default Display Logic**

```javascript
const TEMPLATE_DISPLAY_LOGIC = {
  // Default view for new users
  defaultFilters: {
    complexity: ["simple", "moderate"], 
    showPopular: true,
    maxResults: 9
  },
  
  // Sorting priority
  sortPriority: [
    "popularity",      // Popular templates first
    "complexity",      // Easier templates first 
    "category",        // Group by research domain
    "validation"       // Fully validated templates first
  ],
  
  // Progressive disclosure
  showAdvanced: false, // Require user action to show advanced/complex
  
  // Featured templates (always visible)
  featured: [
    "Python ML Quickstart",
    "R Statistical Analysis", 
    "Jupyter Data Science Stack"
  ]
}
```

---

## 🚀 **Implementation Plan**

### **Phase 1: Enhanced Template Data Model**

```typescript
interface Template {
  // Core identification
  name: string
  description: string
  longDescription?: string
  
  // Difficulty and categorization  
  complexity: "simple" | "moderate" | "advanced" | "complex"
  category: string
  domain: string // "ml" | "datascience" | "bio" | "web" | "base"
  
  // Visual presentation
  icon: string
  color?: string
  popular?: boolean
  featured?: boolean
  
  // Technical metadata
  packages: string[]
  services: string[]
  ports: number[]
  
  // User guidance
  estimatedLaunchTime: number // minutes
  estimatedCost: number // per hour
  prerequisites?: string[]
  learningResources?: string[]
  
  // Advanced metadata
  validationStatus: "validated" | "testing" | "experimental"
  maintainer?: string
  lastUpdated: string
  tags: string[]
}
```

### **Phase 2: Visual Tile Implementation**

1. **Create tile components** with complexity badges and category icons
2. **Implement hover and selection states** with smooth animations
3. **Add responsive grid system** for different screen sizes
4. **Integrate with existing theme system** (all 5 themes)

### **Phase 3: Filtering and Search**

1. **Filter controls** for complexity, category, and popularity
2. **Search functionality** with fuzzy matching on template names and descriptions
3. **Smart sorting** with multiple criteria and user preferences
4. **Progressive disclosure** for advanced templates

### **Phase 4: Enhanced UX Features**

1. **Template preview modal** with detailed information, screenshots, and setup instructions
2. **Beginner guidance system** with tooltips and help text
3. **Template comparison** feature for side-by-side evaluation
4. **User preferences** persistence for filters and sort order

---

## 💡 **User Experience Flows**

### **First-Time User Journey**

```
1. User opens GUI → sees "Popular Beginner Templates" prominently
2. Selects "Python ML Quickstart" → sees detailed preview
3. Clicks "Launch" → guided through simple form with smart defaults
4. Instance launches → user has working ML environment in 3 minutes
```

### **Experienced User Journey**

```
1. User opens GUI → can immediately access all complexity levels
2. Filters by "Advanced + Machine Learning" → sees specialized options
3. Selects custom template → sees detailed configuration options
4. Configures advanced settings → launches with custom parameters
```

### **Progressive Learning Journey**

```
1. User starts with simple templates → builds confidence
2. Sees "Next Steps" suggestions → discovers moderate options
3. Tries moderate templates → gains experience
4. Eventually accesses advanced templates → becomes power user
```

---

## 🎯 **Success Metrics**

### **User Adoption Metrics**
- **Template Selection Time**: Average time from GUI open to template selection
- **Success Rate**: Percentage of launched templates that successfully start
- **User Progression**: Users moving from simple → moderate → advanced over time
- **Popular Templates**: Most frequently selected templates by complexity level

### **User Experience Metrics**  
- **Filter Usage**: How often users filter by complexity/category
- **Search Usage**: Search query patterns and success rates
- **Template Preview**: Modal open rates and time spent viewing details
- **Error Rates**: Failed launches by template complexity level

---

## 🔧 **Technical Considerations**

### **Performance Optimization**
- **Lazy loading** for template icons and preview images
- **Virtual scrolling** for large template lists
- **Caching** of template metadata and user preferences
- **Progressive enhancement** for slower connections

### **Accessibility**
- **Screen reader support** for complexity levels and categories
- **Keyboard navigation** through tile grids
- **High contrast mode** compatibility
- **Focus indicators** for tile selection

### **Testing Strategy**
- **Visual regression tests** for all tile variations across themes
- **Accessibility testing** with axe-core
- **User testing** with researchers of different skill levels
- **Performance testing** with large template catalogs

---

**Total Implementation**: ~2-3 weeks for complete tile system with filtering, search, and progressive disclosure UX.

This enhanced template system will transform the Prism GUI into an intuitive, research-focused platform that guides users from simple to advanced usage while maintaining the professional quality standards.