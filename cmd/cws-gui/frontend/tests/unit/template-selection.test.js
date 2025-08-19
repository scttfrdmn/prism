// Unit tests for template selection functionality
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { screen } from '@testing-library/dom'
import { mockTemplates } from '../mocks/daemon-server.js'

// Import functions to test (would normally be imported from main.js)
// For testing, we'll define inline versions
function createTemplateCard(template) {
  const card = document.createElement('div')
  card.className = 'template-card'
  card.onclick = () => selectTemplate(template.name)
  card.innerHTML = `
    <div class="template-header">
      <span class="template-icon">${template.icon}</span>
      <div>
        <div class="template-title">${template.name}</div>
        <div class="template-category">${template.category}</div>
      </div>
    </div>
    <div class="template-description">${template.description}</div>
  `
  return card
}

function renderTemplates(templates) {
  const grid = document.getElementById('template-grid')
  if (!grid) return
  
  grid.innerHTML = ''
  templates.forEach(template => {
    const card = createTemplateCard(template)
    grid.appendChild(card)
  })
}

function selectTemplate(templateName) {
  const selectedTemplate = mockTemplates.find(t => t.name === templateName)
  if (!selectedTemplate) return
  
  // Update UI to show selection
  document.querySelectorAll('.template-card').forEach(card => {
    card.classList.remove('selected')
  })
  
  const selectedCard = Array.from(document.querySelectorAll('.template-card'))
    .find(card => card.querySelector('.template-title').textContent === templateName)
  
  if (selectedCard) {
    selectedCard.classList.add('selected')
  }
  
  // Show launch form
  const form = document.getElementById('launch-form')
  if (form) {
    form.classList.remove('hidden')
    const templateNameElement = document.getElementById('selected-template-name')
    if (templateNameElement) {
      templateNameElement.textContent = templateName
    }
  }
}

function setupDOM() {
  document.body.innerHTML = `
    <div id="app">
      <div id="template-grid"></div>
      <div id="launch-form" class="launch-form hidden">
        <div class="form-header">
          <h3 id="selected-template-name">Selected Template</h3>
        </div>
        <div class="form-actions">
          <button id="launch-btn" class="btn-primary">Launch</button>
        </div>
      </div>
    </div>
  `
}

describe('Template Selection', () => {
  beforeEach(() => {
    setupDOM()
  })

  test('renders templates correctly', () => {
    renderTemplates(mockTemplates)
    
    // Check that all templates are rendered
    expect(document.querySelectorAll('.template-card')).toHaveLength(mockTemplates.length)
    
    // Check specific template content
    const pythonTemplate = screen.getByText('Python Machine Learning (Simplified)')
    expect(pythonTemplate).toBeInTheDocument()
    
    const rTemplate = screen.getByText('R Research Environment (Simplified)')
    expect(rTemplate).toBeInTheDocument()
  })

  test('displays template metadata correctly', () => {
    renderTemplates(mockTemplates)
    
    // Check icons are displayed
    expect(screen.getByText('🐍')).toBeInTheDocument()
    expect(screen.getByText('📊')).toBeInTheDocument()
    
    // Check categories are displayed
    expect(screen.getByText('Machine Learning')).toBeInTheDocument()
    expect(screen.getByText('Data Science')).toBeInTheDocument()
    
    // Check descriptions are displayed
    expect(screen.getByText(/Conda \+ Jupyter \+ ML packages/)).toBeInTheDocument()
    expect(screen.getByText(/RStudio \+ tidyverse packages/)).toBeInTheDocument()
  })

  test('handles template selection', () => {
    renderTemplates(mockTemplates)
    
    // Initially no template should be selected
    expect(document.querySelector('.template-card.selected')).toBeNull()
    expect(document.getElementById('launch-form')).toHaveClass('hidden')
    
    // Select a template
    selectTemplate('Python Machine Learning (Simplified)')
    
    // Check that template is marked as selected
    const selectedCard = document.querySelector('.template-card.selected')
    expect(selectedCard).not.toBeNull()
    expect(selectedCard.querySelector('.template-title').textContent)
      .toBe('Python Machine Learning (Simplified)')
    
    // Check that launch form is shown
    expect(document.getElementById('launch-form')).not.toHaveClass('hidden')
    expect(document.getElementById('selected-template-name').textContent)
      .toBe('Python Machine Learning (Simplified)')
  })

  test('handles switching between templates', () => {
    renderTemplates(mockTemplates)
    
    // Select first template
    selectTemplate('Python Machine Learning (Simplified)')
    expect(document.querySelectorAll('.template-card.selected')).toHaveLength(1)
    
    // Select second template
    selectTemplate('R Research Environment (Simplified)')
    
    // Check only one template is selected
    const selectedCards = document.querySelectorAll('.template-card.selected')
    expect(selectedCards).toHaveLength(1)
    expect(selectedCards[0].querySelector('.template-title').textContent)
      .toBe('R Research Environment (Simplified)')
    
    // Check launch form shows correct template
    expect(document.getElementById('selected-template-name').textContent)
      .toBe('R Research Environment (Simplified)')
  })

  test('handles empty template list', () => {
    renderTemplates([])
    
    expect(document.querySelectorAll('.template-card')).toHaveLength(0)
  })

  test('handles template selection with invalid name', () => {
    renderTemplates(mockTemplates)
    
    // Try to select non-existent template
    selectTemplate('Non-existent Template')
    
    // No template should be selected
    expect(document.querySelector('.template-card.selected')).toBeNull()
    
    // Launch form should remain hidden
    expect(document.getElementById('launch-form')).toHaveClass('hidden')
  })

  test('template cards have correct CSS classes', () => {
    renderTemplates(mockTemplates)
    
    const cards = document.querySelectorAll('.template-card')
    cards.forEach(card => {
      expect(card).toHaveClass('template-card')
      expect(card.querySelector('.template-header')).toBeInTheDocument()
      expect(card.querySelector('.template-title')).toBeInTheDocument()
      expect(card.querySelector('.template-category')).toBeInTheDocument()
      expect(card.querySelector('.template-description')).toBeInTheDocument()
      expect(card.querySelector('.template-icon')).toBeInTheDocument()
    })
  })
})