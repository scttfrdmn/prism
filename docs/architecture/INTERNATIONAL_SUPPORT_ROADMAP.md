# International Support Roadmap

**Last Updated**: October 2025
**Status**: Planning
**Target**: v0.7.0-0.8.0 (2026-2027)

---

## Executive Summary

Prism's current architecture is US-centric, limiting its utility for international research collaborations and non-English-speaking research communities. This document outlines a comprehensive roadmap for international support across infrastructure, localization, and accessibility dimensions.

---

## 1. AWS Regional Expansion

### Current State (v0.5.5)

**Limited Regional Support**:
```bash
# Currently hardcoded US region mappings
AMI_MAPPINGS = {
    "ubuntu-22.04": {
        "us-east-1": "ami-0c55b159cbfafe1f0",
        "us-west-1": "ami-0d9858aa3c6322f73",
        "us-west-2": "ami-0928f4202481dfdf6",
        # No non-US regions
    }
}
```

**Problems**:
- ❌ European researchers experience high latency
- ❌ Asia-Pacific regions not supported
- ❌ Data residency requirements unmet (GDPR, etc.)
- ❌ Higher cross-region data transfer costs

### Phase 1: Global AMI Discovery (v0.6.0 - Q2 2026)

**Intelligent Regional AMI Resolution**:
```go
// pkg/ami/resolver.go
type RegionalAMIResolver struct {
    // Use AWS SSM Parameter Store for automatic AMI discovery
    ssmClient *ssm.Client
}

func (r *RegionalAMIResolver) ResolveAMI(distro string, version string, region string) (string, error) {
    // Query AWS-managed SSM parameters for latest AMIs
    parameter := fmt.Sprintf("/aws/service/canonical/ubuntu/server/%s/stable/current/amd64/hvm/ebs-gp2/ami-id", version)

    result, err := r.ssmClient.GetParameter(&ssm.GetParameterInput{
        Name: aws.String(parameter),
    })

    if err != nil {
        return "", fmt.Errorf("AMI not available in region %s: %w", region, err)
    }

    return *result.Parameter.Value, nil
}
```

**Supported Regions (Priority Order)**:

**Tier 1: High Priority (v0.6.0)**:
- `eu-west-1` (Ireland) - GDPR-compliant, European research hub
- `eu-central-1` (Frankfurt) - German research institutions
- `ap-southeast-1` (Singapore) - Asia-Pacific research
- `ap-northeast-1` (Tokyo) - Japanese research institutions
- `ca-central-1` (Canada) - Canadian research organizations

**Tier 2: Medium Priority (v0.6.1)**:
- `eu-west-2` (London) - UK research post-Brexit
- `eu-north-1` (Stockholm) - Nordic research institutions
- `ap-southeast-2` (Sydney) - Australian research organizations
- `ap-south-1` (Mumbai) - Indian research institutions
- `sa-east-1` (São Paulo) - Latin American research

**Tier 3: Extended Coverage (v0.7.0)**:
- `eu-west-3` (Paris) - French research institutions
- `eu-south-1` (Milan) - Southern European research
- `ap-northeast-2` (Seoul) - Korean research institutions
- `ap-northeast-3` (Osaka) - Additional Japanese capacity
- `me-south-1` (Bahrain) - Middle Eastern research
- `af-south-1` (Cape Town) - African research institutions

**Implementation**:
```bash
# Automatic regional AMI discovery
prism ami discover --region eu-west-1 --distro ubuntu-22.04

🔍 Discovering AMI for ubuntu-22.04 in eu-west-1...
✅ Found: ami-0d7892b35e6d2e2e9
📋 Source: AWS SSM Parameter Store
🔐 Verified: AWS-managed, security-patched

# Launch in any supported region
prism launch python-ml european-analysis \
  --region eu-west-1 \
  --data-residency eu

# Automatic region selection based on data location
prism launch bioinformatics genomics-eu \
  --data-location s3://eu-genomics-data/ \
  --auto-select-region

🔍 Analyzing data location...
✅ Data in eu-west-1
✅ Launching instance in eu-west-1 (minimize transfer costs)
```

### Phase 2: Data Residency Compliance (v0.6.1 - Q3 2026)

**GDPR and Data Sovereignty Support**:
```yaml
# templates/python-ml-eu.yml
name: "Python ML (EU/GDPR Compliant)"
data_residency:
  regions: ["eu-west-1", "eu-central-1", "eu-north-1"]
  prohibit_transfer: true
  encryption_required: true
  audit_logging: comprehensive

compliance:
  gdpr: true
  data_processing_agreement: true
  right_to_erasure: true
  data_portability: true

security:
  encryption_at_rest: true
  encryption_in_transit: true
  key_management: "aws-kms-eu"
```

**Compliance Commands**:
```bash
# Enforce data residency
prism admin policy set data-residency \
  --project eu-research \
  --allowed-regions eu-west-1,eu-central-1 \
  --prohibit-cross-region-transfer \
  --require-encryption

# GDPR compliance reporting
prism admin gdpr report --project eu-research

GDPR Compliance Report:
✅ Data residency: All data in EU regions
✅ Encryption: 100% of storage encrypted
✅ Audit logging: Complete access trail
✅ Right to erasure: Implemented via `prism project delete`
✅ Data portability: Export available via `prism project export`
⚠️  Data Processing Agreement: Requires manual acceptance

Next Steps:
→ Review DPA at: https://prism.io/dpa
→ Accept with: prism admin gdpr accept-dpa --project eu-research
```

---

## 2. Internationalization (i18n)

### Current State (v0.5.5)

- ❌ English-only interface (CLI, TUI, GUI)
- ❌ English-only documentation
- ❌ English-only error messages
- ❌ US date/currency formats hardcoded

### Phase 1: Infrastructure (v0.7.0 - Q4 2026)

**i18n Framework Setup**:
```go
// pkg/i18n/translator.go
package i18n

import (
    "golang.org/x/text/language"
    "golang.org/x/text/message"
)

var SupportedLanguages = []language.Tag{
    language.English,      // en
    language.Spanish,      // es
    language.French,       // fr
    language.German,       // de
    language.Japanese,     // ja
    language.Chinese,      // zh
    language.Korean,       // ko
    language.Portuguese,   // pt
    language.Italian,      // it
    language.Dutch,        // nl
}

type Translator struct {
    printer *message.Printer
}

func NewTranslator(lang string) *Translator {
    tag := language.Make(lang)
    return &Translator{
        printer: message.NewPrinter(tag),
    }
}

func (t *Translator) T(key string, args ...interface{}) string {
    return t.printer.Sprintf(key, args...)
}
```

**Message Catalogs**:
```
locales/
├── en/
│   ├── messages.json
│   └── errors.json
├── es/
│   ├── messages.json
│   └── errors.json
├── fr/
│   ├── messages.json
│   └── errors.json
├── de/
│   ├── messages.json
│   └── errors.json
├── ja/
│   ├── messages.json
│   └── errors.json
└── zh/
    ├── messages.json
    └── errors.json
```

**Example Messages**:
```json
// locales/en/messages.json
{
  "launch.success": "✅ Launched instance %s successfully",
  "launch.progress": "🚀 Launching instance %s in region %s...",
  "cost.estimate": "💰 Estimated cost: $%0.2f/hour",
  "error.no_template": "❌ Template '%s' not found"
}

// locales/es/messages.json
{
  "launch.success": "✅ Instancia %s lanzada exitosamente",
  "launch.progress": "🚀 Lanzando instancia %s en región %s...",
  "cost.estimate": "💰 Costo estimado: $%0.2f/hora",
  "error.no_template": "❌ Plantilla '%s' no encontrada"
}

// locales/ja/messages.json
{
  "launch.success": "✅ インスタンス %s を正常に起動しました",
  "launch.progress": "🚀 リージョン %s でインスタンス %s を起動中...",
  "cost.estimate": "💰 推定コスト: $%0.2f/時間",
  "error.no_template": "❌ テンプレート '%s' が見つかりません"
}
```

### Phase 2: CLI/TUI Localization (v0.7.1 - Q1 2027)

**Language Selection**:
```bash
# Set user language preference
prism config set language ja
prism config set region ap-northeast-1
prism config set currency JPY

# Launch with localized output
prism launch python-ml 分析プロジェクト

🔍 AMI を解決中: ubuntu-22.04 (ap-northeast-1)
✅ 検出: ami-0d7892b35e6d2e2e9
🚀 インスタンスを起動中: 分析プロジェクト
📊 推定コスト: ¥45.50/時間
✅ 起動成功: i-0123456789abcdef0
```

**Automatic Language Detection**:
```bash
# Detect from environment
export LANG=es_ES.UTF-8
prism templates

Plantillas Disponibles:
├── python-ml: Aprendizaje automático con Python
├── r-research: Investigación estadística con R
├── bioinformatics: Análisis bioinformático
└── web-dev: Desarrollo web

# Override with flag
prism templates --lang en
```

### Phase 3: GUI Localization (v0.7.2 - Q2 2027)

**React i18n Integration**:
```typescript
// cmd/cws-gui/frontend/src/i18n.ts
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

i18n
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: require('./locales/en.json') },
      es: { translation: require('./locales/es.json') },
      fr: { translation: require('./locales/fr.json') },
      de: { translation: require('./locales/de.json') },
      ja: { translation: require('./locales/ja.json') },
      zh: { translation: require('./locales/zh.json') },
    },
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false,
    },
  });

export default i18n;
```

**Cloudscape Components**:
```typescript
// All Cloudscape components support RTL and i18n
import { Button, Header } from '@cloudscape-design/components';
import { useTranslation } from 'react-i18next';

function LaunchButton() {
  const { t } = useTranslation();

  return (
    <Button variant="primary">
      {t('launch.button')} {/* "Launch Instance" or "起動" */}
    </Button>
  );
}
```

### Phase 4: Documentation Localization (v0.7.3 - Q3 2027)

**Multi-Language Documentation**:
```
docs/
├── en/              # English (primary)
│   ├── index.md
│   ├── user-guides/
│   └── architecture/
├── es/              # Spanish
│   ├── index.md
│   └── guías-de-usuario/
├── fr/              # French
│   ├── index.md
│   └── guides-utilisateur/
├── de/              # German
│   ├── index.md
│   └── benutzerhandbücher/
├── ja/              # Japanese
│   ├── index.md
│   └── ユーザーガイド/
└── zh/              # Chinese
    ├── index.md
    └── 用户指南/
```

**MkDocs Multilingual**:
```yaml
# mkdocs.yml
plugins:
  - i18n:
      default_language: en
      languages:
        en:
          name: English
          site_name: Prism Documentation
        es:
          name: Español
          site_name: Documentación de Prism
        fr:
          name: Français
          site_name: Documentation de Prism
        de:
          name: Deutsch
          site_name: Prism-Dokumentation
        ja:
          name: 日本語
          site_name: Prism ドキュメント
        zh:
          name: 中文
          site_name: Prism 文档
```

---

## 3. Currency and Formatting

### Current State (v0.5.5)

- ❌ USD-only cost display
- ❌ US date formats (MM/DD/YYYY)
- ❌ No currency conversion

### Phase 1: Regional Formatting (v0.7.0 - Q4 2026)

**Locale-Aware Formatting**:
```go
// pkg/format/currency.go
type CurrencyFormatter struct {
    locale   string
    currency string
}

func (cf *CurrencyFormatter) FormatCost(usd float64) string {
    switch cf.currency {
    case "JPY":
        jpy := usd * 150.0  // Exchange rate from API
        return fmt.Sprintf("¥%0.0f", jpy)
    case "EUR":
        eur := usd * 0.92
        return fmt.Sprintf("€%0.2f", eur)
    case "GBP":
        gbp := usd * 0.79
        return fmt.Sprintf("£%0.2f", gbp)
    default:
        return fmt.Sprintf("$%0.2f", usd)
    }
}
```

**Usage**:
```bash
# Japanese researcher sees costs in JPY
prism cost estimate python-ml --region ap-northeast-1

インスタンスコストの見積もり:
├── コンピューティング: ¥6,825/時間
├── ストレージ (100 GB): ¥1,500/月
├── データ転送: ¥1,125/GB
└── 合計見積もり: ¥205,000/月

# European researcher sees EUR
prism cost estimate python-ml --region eu-west-1

Instance Cost Estimate:
├── Compute: €4.14/hour
├── Storage (100 GB): €9.20/month
├── Data Transfer: €0.08/GB
└── Total Estimate: €120.00/month
```

**Exchange Rate API**:
```bash
# Automatic exchange rate updates
prism admin exchange-rates update

Updating exchange rates from ECB...
✅ EUR: 0.92 USD
✅ GBP: 0.79 USD
✅ JPY: 150.00 USD
✅ CAD: 1.35 USD
✅ AUD: 1.52 USD
Last updated: 2025-10-19 14:32 UTC
```

---

## 4. Accessibility (a11y)

### Current State (v0.5.5)

- ✅ GUI uses Cloudscape Design System (WCAG AA compliant)
- ⚠️  CLI lacks screen reader support
- ⚠️  TUI has limited keyboard navigation hints

### Phase 1: Enhanced CLI Accessibility (v0.7.0 - Q4 2026)

**Screen Reader Support**:
```bash
# Verbose mode for screen readers
prism launch python-ml my-project --accessible

Launching instance my-project
Step 1 of 5: Resolving AMI for ubuntu-22.04
Status: In progress
[Progress indicator: 20%]
Step 2 of 5: Creating security group
Status: In progress
[Progress indicator: 40%]
...
```

**Alternative Output Formats**:
```bash
# JSON output for assistive tools
prism list --format json | jq

# Plain text without unicode symbols
prism list --no-emoji --no-colors

Instances:
  my-project
    Status: running
    Type: t3.medium
    IP: 54.123.45.67
    Cost: 0.0416 USD/hour
```

### Phase 2: TUI Accessibility (v0.7.1 - Q1 2027)

**Enhanced Keyboard Navigation**:
```
TUI Keyboard Shortcuts:
- Tab: Next field
- Shift+Tab: Previous field
- Enter: Select/Activate
- Esc: Cancel/Go back
- Ctrl+H: Help overlay
- Ctrl+N: Navigation hints
- Ctrl+R: Screen reader mode

Screen Reader Announcements:
"Templates list. 4 items. Currently selected: Python ML.
Press Enter to launch, Tab to move to next item."
```

### Phase 3: High Contrast and Large Text (v0.7.2 - Q2 2027)

**GUI Accessibility Settings**:
```typescript
// GUI Settings Panel
<FormField label="Accessibility">
  <Select
    options={[
      { label: "Standard", value: "standard" },
      { label: "High Contrast", value: "high-contrast" },
      { label: "Large Text", value: "large-text" },
      { label: "High Contrast + Large Text", value: "hc-large" }
    ]}
  />
</FormField>
```

---

## 5. Right-to-Left (RTL) Language Support

### Phase 1: RTL Infrastructure (v0.8.0 - Q3 2027)

**Arabic and Hebrew Support**:
```typescript
// GUI automatically detects RTL languages
import { applyMode, Mode } from '@cloudscape-design/global-styles';

function App() {
  const { i18n } = useTranslation();
  const isRTL = ['ar', 'he'].includes(i18n.language);

  useEffect(() => {
    applyMode(isRTL ? Mode.Dark : Mode.Light);
    document.dir = isRTL ? 'rtl' : 'ltr';
  }, [isRTL]);

  return <CloudscapeAppLayout />;
}
```

**CLI RTL Support**:
```bash
# Arabic interface
export LANG=ar_SA.UTF-8
prism launch python-ml مشروع-التحليل

🔍 جارٍ حل AMI: ubuntu-22.04 (me-south-1)
✅ تم العثور على: ami-0d7892b35e6d2e2e9
🚀 جارٍ تشغيل المثيل: مشروع-التحليل
💰 التكلفة المقدرة: 0.05 دولار/ساعة
✅ تم التشغيل بنجاح: i-0123456789abcdef0
```

---

## 6. Compliance and Legal

### Phase 1: International Data Protection (v0.6.0 - Q2 2026)

**GDPR Compliance (EU)**:
- ✅ Data residency enforcement
- ✅ Right to erasure
- ✅ Data portability
- ✅ Audit logging
- ✅ Privacy by design

**PIPEDA Compliance (Canada)**:
- ✅ Consent management
- ✅ Data access controls
- ✅ Breach notification
- ✅ Cross-border transfer rules

**APPI Compliance (Japan)**:
- ✅ Personal information protection
- ✅ Cross-border data transfer notifications
- ✅ Third-party oversight

### Phase 2: Export Control (v0.6.1 - Q3 2026)

**Technical Data Controls**:
```bash
# Automatic export control checks
prism launch high-performance-computing quantum-research \
  --region us-west-2

⚠️  Export Control Warning:
Template "high-performance-computing" may contain technical data
subject to US Export Administration Regulations (EAR).

User citizenship: India
Destination region: us-west-2 (allowed)
Technical data classification: EAR99

✅ Launch authorized
📋 Logged for compliance review
```

---

## 7. Community Translation Program

### Phase 1: Volunteer Translators (v0.7.0 - Q4 2026)

**Translation Workflow**:
```
1. English source strings updated
2. Automated extraction to translation files
3. Community translators contribute via Crowdin
4. Research IT teams review translations
5. Approved translations merged to release
```

**Contributor Recognition**:
```bash
prism about --credits

Prism v0.7.0

Core Team:
[...]

Translation Contributors:
├── Spanish (es): Maria Garcia, Carlos Rodriguez
├── French (fr): Pierre Dubois, Sophie Martin
├── German (de): Hans Mueller, Anna Schmidt
├── Japanese (ja): Yuki Tanaka, Kenji Yamamoto
├── Chinese (zh): Wei Zhang, Li Wang
└── Korean (ko): Min-jun Kim, Ji-woo Park

Thank you to our global community! 🌍
```

---

## 8. Implementation Priority

### High Priority (2026)
1. ✅ **Global AMI Discovery** - Critical for international users
2. ✅ **EU Region Support** - GDPR compliance, large research community
3. ✅ **Basic i18n Infrastructure** - Foundation for all translations

### Medium Priority (2027)
4. ✅ **Spanish/French/German Localization** - Large language communities
5. ✅ **Japanese/Chinese Localization** - Asia-Pacific research
6. ✅ **Currency Formatting** - Cost transparency
7. ✅ **Enhanced Accessibility** - Inclusive design

### Lower Priority (2028+)
8. ✅ **RTL Language Support** - Arabic, Hebrew
9. ✅ **Additional Languages** - Korean, Portuguese, Dutch, etc.
10. ✅ **Voice Interface** - Accessibility innovation

---

## Success Metrics

### Adoption
- **30%+ international users** (non-US) within 12 months
- **15+ countries** with active Prism deployments
- **5+ languages** with >80% translation coverage

### Quality
- **WCAG AA compliance** across all interfaces
- **<5% translation errors** reported
- **95%+ user satisfaction** from international users

### Community
- **50+ volunteer translators** contributing
- **10+ research institutions** in non-English-speaking countries
- **International template marketplace** with multi-language templates

---

**Prism International**: Research computing without borders. Support global collaboration while respecting data sovereignty and cultural diversity.
