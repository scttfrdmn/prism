<template>
  <div class="batch-invitation-view">
    <h1>Batch Invitation Management</h1>
    
    <div class="operation-tabs">
      <div 
        v-for="tab in tabs" 
        :key="tab.id"
        :class="{ 'tab': true, 'active': activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        {{ tab.label }}
      </div>
    </div>
    
    <!-- Create Invitations Tab -->
    <div v-if="activeTab === 'create'" class="tab-content">
      <div class="operation-panel">
        <h2>Create Batch Invitations</h2>
        <p>Create multiple invitations at once from a CSV file.</p>
        
        <div class="form-group">
          <label>CSV File:</label>
          <div class="file-input-row">
            <input type="text" v-model="createParams.filePath" placeholder="Select CSV file..." readonly />
            <button @click="selectImportFile" class="secondary-button">Browse</button>
          </div>
        </div>
        
        <div class="form-group">
          <label>S3 Config Path (optional):</label>
          <input type="text" v-model="createParams.s3ConfigPath" placeholder="s3://bucket/path" />
        </div>
        
        <div class="form-group">
          <label>Parent Token (optional):</label>
          <input type="text" v-model="createParams.parentToken" placeholder="inv-..." />
        </div>
        
        <div class="form-group">
          <label>CSV has header row:</label>
          <input type="checkbox" v-model="createParams.hasHeader" />
        </div>
        
        <div class="form-group">
          <label>Concurrency:</label>
          <input type="number" v-model="createParams.concurrency" min="1" max="20" />
        </div>
        
        <div class="form-group">
          <label>Output File:</label>
          <div class="file-input-row">
            <input type="text" v-model="createParams.outputFile" placeholder="Select output file..." readonly />
            <button @click="selectExportFile('create')" class="secondary-button">Browse</button>
          </div>
        </div>
        
        <div class="button-row">
          <button @click="createTemplate" class="secondary-button">Generate Template</button>
          <button @click="createInvitations" class="primary-button">Create Invitations</button>
        </div>
        
        <div v-if="csvPreview.length > 0" class="preview-panel">
          <h3>CSV Preview</h3>
          <table class="data-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Valid Days</th>
                <th>Device Bound</th>
                <th>Max Devices</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in csvPreview" :key="index">
                <td>{{ row.name }}</td>
                <td>{{ row.type }}</td>
                <td>{{ row.validDays }}</td>
                <td>{{ row.deviceBound ? 'Yes' : 'No' }}</td>
                <td>{{ row.maxDevices }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
    
    <!-- Export Invitations Tab -->
    <div v-if="activeTab === 'export'" class="tab-content">
      <div class="operation-panel">
        <h2>Export Invitations</h2>
        <p>Export all current invitations to a CSV file.</p>
        
        <div class="form-group">
          <label>Output File:</label>
          <div class="file-input-row">
            <input type="text" v-model="exportParams.outputFile" placeholder="Select output file..." readonly />
            <button @click="selectExportFile('export')" class="secondary-button">Browse</button>
          </div>
        </div>
        
        <div class="button-row">
          <button @click="exportInvitations" class="primary-button">Export Invitations</button>
        </div>
        
        <div v-if="allInvitations.length > 0" class="preview-panel">
          <h3>Current Invitations</h3>
          <table class="data-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Expires In</th>
                <th>Device Bound</th>
                <th>Max Devices</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(inv, index) in allInvitations" :key="index">
                <td>{{ inv.name }}</td>
                <td>{{ inv.type }}</td>
                <td>{{ inv.expiresIn }}</td>
                <td>{{ inv.deviceBound ? 'Yes' : 'No' }}</td>
                <td>{{ inv.maxDevices }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
    
    <!-- Accept Invitations Tab -->
    <div v-if="activeTab === 'accept'" class="tab-content">
      <div class="operation-panel">
        <h2>Accept Batch Invitations</h2>
        <p>Accept multiple invitations from a CSV file.</p>
        
        <div class="form-group">
          <label>CSV File:</label>
          <div class="file-input-row">
            <input type="text" v-model="acceptParams.filePath" placeholder="Select CSV file..." readonly />
            <button @click="selectAcceptFile" class="secondary-button">Browse</button>
          </div>
        </div>
        
        <div class="form-group">
          <label>Name Prefix (optional):</label>
          <input type="text" v-model="acceptParams.namePrefix" placeholder="Project-" />
        </div>
        
        <div class="form-group">
          <label>CSV has header row:</label>
          <input type="checkbox" v-model="acceptParams.hasHeader" />
        </div>
        
        <div class="button-row">
          <button @click="acceptInvitations" class="primary-button">Accept Invitations</button>
        </div>
      </div>
    </div>
    
    <!-- Results Panel -->
    <div v-if="lastResult" class="results-panel">
      <h3>Operation Results</h3>
      <div class="result-summary">
        <p><strong>Operation:</strong> {{ operationName(lastResult.operation) }}</p>
        <p><strong>Total Processed:</strong> {{ lastResult.totalProcessed }}</p>
        <p><strong>Successful:</strong> {{ lastResult.totalSuccessful }}</p>
        <p><strong>Failed:</strong> {{ lastResult.totalFailed }}</p>
        <p v-if="lastResult.outputFile"><strong>Output File:</strong> {{ lastResult.outputFile }}</p>
        <p v-if="lastResult.error" class="error-message">{{ lastResult.error }}</p>
      </div>
      
      <div class="result-actions" v-if="lastResult.outputFile">
        <button @click="openCSVFile(lastResult.outputFile)" class="secondary-button">Open CSV</button>
        <button @click="openCSVFolder(lastResult.outputFile)" class="secondary-button">Open Folder</button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "BatchInvitationView",
  data() {
    return {
      activeTab: "create",
      tabs: [
        { id: "create", label: "Create Invitations" },
        { id: "export", label: "Export Invitations" },
        { id: "accept", label: "Accept Invitations" }
      ],
      createParams: {
        filePath: "",
        s3ConfigPath: "",
        parentToken: "",
        hasHeader: true,
        concurrency: 5,
        outputFile: ""
      },
      exportParams: {
        outputFile: ""
      },
      acceptParams: {
        filePath: "",
        namePrefix: "",
        hasHeader: true
      },
      csvPreview: [],
      allInvitations: [],
      lastResult: null,
      isLoading: false,
      error: ""
    };
  },
  mounted() {
    // Load existing invitations
    this.loadAllInvitations();
    
    // Load last operation result if available
    this.loadLastResult();
  },
  methods: {
    async selectImportFile() {
      try {
        const filePath = await window.go.pages.BatchInvitationPage.SelectImportFile();
        if (filePath) {
          this.createParams.filePath = filePath;
          this.previewCSVFile();
        }
      } catch (error) {
        this.error = `Error selecting file: ${error}`;
      }
    },
    
    async selectExportFile(type) {
      try {
        const filePath = await window.go.pages.BatchInvitationPage.SelectExportFile();
        if (filePath) {
          if (type === 'create') {
            this.createParams.outputFile = filePath;
          } else if (type === 'export') {
            this.exportParams.outputFile = filePath;
          }
        }
      } catch (error) {
        this.error = `Error selecting file: ${error}`;
      }
    },
    
    async selectAcceptFile() {
      try {
        const filePath = await window.go.pages.BatchInvitationPage.SelectImportFile();
        if (filePath) {
          this.acceptParams.filePath = filePath;
        }
      } catch (error) {
        this.error = `Error selecting file: ${error}`;
      }
    },
    
    async previewCSVFile() {
      if (!this.createParams.filePath) return;
      
      try {
        this.isLoading = true;
        const response = await window.go.pages.BatchInvitationPage.PreviewCSVFile(
          this.createParams.filePath, 
          this.createParams.hasHeader
        );
        
        const data = JSON.parse(response);
        if (data.success === false) {
          this.error = data.error;
          this.csvPreview = [];
        } else {
          this.csvPreview = data;
          this.error = "";
        }
      } catch (error) {
        this.error = `Error previewing CSV file: ${error}`;
        this.csvPreview = [];
      } finally {
        this.isLoading = false;
      }
    },
    
    async createInvitations() {
      if (!this.createParams.filePath) {
        this.error = "Please select a CSV file";
        return;
      }
      
      try {
        this.isLoading = true;
        const response = await window.go.pages.BatchInvitationPage.CreateInvitations(
          this.createParams.filePath,
          this.createParams.s3ConfigPath,
          this.createParams.parentToken,
          this.createParams.hasHeader,
          this.createParams.concurrency,
          this.createParams.outputFile
        );
        
        const data = JSON.parse(response);
        if (data.success === false) {
          this.error = data.error;
        } else {
          this.lastResult = data;
          this.error = "";
          // Refresh invitation list
          this.loadAllInvitations();
        }
      } catch (error) {
        this.error = `Error creating invitations: ${error}`;
      } finally {
        this.isLoading = false;
      }
    },
    
    async exportInvitations() {
      if (!this.exportParams.outputFile) {
        this.error = "Please select an output file";
        return;
      }
      
      try {
        this.isLoading = true;
        const response = await window.go.pages.BatchInvitationPage.ExportAllInvitations(
          this.exportParams.outputFile
        );
        
        const data = JSON.parse(response);
        if (data.success === false) {
          this.error = data.error;
        } else {
          this.lastResult = data;
          this.error = "";
        }
      } catch (error) {
        this.error = `Error exporting invitations: ${error}`;
      } finally {
        this.isLoading = false;
      }
    },
    
    async acceptInvitations() {
      if (!this.acceptParams.filePath) {
        this.error = "Please select a CSV file";
        return;
      }
      
      try {
        this.isLoading = true;
        const response = await window.go.pages.BatchInvitationPage.AcceptInvitations(
          this.acceptParams.filePath,
          this.acceptParams.namePrefix,
          this.acceptParams.hasHeader
        );
        
        const data = JSON.parse(response);
        if (data.success === false) {
          this.error = data.error;
        } else {
          this.lastResult = data;
          this.error = "";
        }
      } catch (error) {
        this.error = `Error accepting invitations: ${error}`;
      } finally {
        this.isLoading = false;
      }
    },
    
    async createTemplate() {
      try {
        const response = await window.go.pages.BatchInvitationPage.GenerateCSVTemplate();
        const data = JSON.parse(response);
        if (data.success === false) {
          this.error = data.error;
        } else {
          // Template was created and opened
          this.error = "";
        }
      } catch (error) {
        this.error = `Error creating template: ${error}`;
      }
    },
    
    async loadAllInvitations() {
      try {
        const response = await window.go.pages.BatchInvitationPage.GetAllInvitations();
        const data = JSON.parse(response);
        if (data.success === false) {
          this.error = data.error;
          this.allInvitations = [];
        } else {
          this.allInvitations = data;
        }
      } catch (error) {
        this.error = `Error loading invitations: ${error}`;
        this.allInvitations = [];
      }
    },
    
    async loadLastResult() {
      try {
        const response = await window.go.pages.BatchInvitationPage.GetLastOperationResult();
        const data = JSON.parse(response);
        if (data.success !== false) {
          this.lastResult = data;
        }
      } catch (error) {
        // Ignore errors, no last result available
      }
    },
    
    async openCSVFile(filePath) {
      try {
        await window.go.pages.BatchInvitationPage.OpenCSVFile(filePath);
      } catch (error) {
        this.error = `Error opening file: ${error}`;
      }
    },
    
    async openCSVFolder(filePath) {
      try {
        await window.go.pages.BatchInvitationPage.OpenCSVFolder(filePath);
      } catch (error) {
        this.error = `Error opening folder: ${error}`;
      }
    },
    
    operationName(operation) {
      const names = {
        "create": "Create Invitations",
        "export": "Export Invitations",
        "accept": "Accept Invitations"
      };
      return names[operation] || operation;
    }
  },
  watch: {
    'createParams.hasHeader'() {
      // Refresh preview when header setting changes
      if (this.createParams.filePath) {
        this.previewCSVFile();
      }
    }
  }
}
</script>

<style scoped>
.batch-invitation-view {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

h1 {
  margin-bottom: 30px;
  color: #333;
}

.operation-tabs {
  display: flex;
  border-bottom: 1px solid #ddd;
  margin-bottom: 20px;
}

.tab {
  padding: 12px 20px;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  font-weight: 500;
  transition: all 0.3s;
}

.tab:hover {
  background-color: #f5f5f5;
}

.tab.active {
  border-bottom: 2px solid #0078d7;
  color: #0078d7;
}

.tab-content {
  margin-bottom: 30px;
}

.operation-panel {
  background-color: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.form-group {
  margin-bottom: 15px;
}

label {
  display: block;
  margin-bottom: 5px;
  font-weight: 500;
}

input[type="text"],
input[type="number"] {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
}

.file-input-row {
  display: flex;
  gap: 10px;
}

.file-input-row input {
  flex: 1;
}

.button-row {
  margin-top: 25px;
  display: flex;
  gap: 15px;
  justify-content: flex-end;
}

.primary-button {
  background-color: #0078d7;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 500;
}

.primary-button:hover {
  background-color: #006cc1;
}

.secondary-button {
  background-color: #f0f0f0;
  color: #333;
  border: 1px solid #ddd;
  padding: 8px 15px;
  border-radius: 4px;
  cursor: pointer;
}

.secondary-button:hover {
  background-color: #e0e0e0;
}

.preview-panel {
  margin-top: 30px;
  border-top: 1px solid #eee;
  padding-top: 20px;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 10px;
}

.data-table th, 
.data-table td {
  padding: 8px 12px;
  text-align: left;
  border-bottom: 1px solid #eee;
}

.data-table th {
  background-color: #f5f5f5;
  font-weight: 500;
}

.results-panel {
  background-color: #f9f9f9;
  border-radius: 8px;
  padding: 20px;
  margin-top: 30px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.result-summary {
  margin-bottom: 20px;
}

.result-summary p {
  margin: 8px 0;
}

.result-actions {
  display: flex;
  gap: 10px;
}

.error-message {
  color: #d32f2f;
  font-weight: 500;
}
</style>