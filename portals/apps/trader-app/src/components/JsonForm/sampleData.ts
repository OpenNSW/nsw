// Realistic sample data for auto-filling forms
// Maps field names to realistic values based on the form schemas

export interface SampleDataMap {
  [fieldName: string]: unknown;
}

// Main sample data lookup map
export const SAMPLE_DATA_MAP: SampleDataMap = {
  // General Information form
  consigneeName: 'British Wellness Imports Ltd.',
  consigneeAddress: '45 Trade Centre Avenue\nLondon, E14 5HP\nUnited Kingdom',
  countryOfDestination: 'GB',

  // Health Certificate form
  productDescription: 'Organic Desiccated Coconut (Fine Grade)',
  batchLotNumbers: 'DC-2026-JAN-05',
  productionExpiryDates: 'Production: 2026-01-15, Expiry: 2027-01-15',
  microbiologicalTestReportId: 'ITI/2026/LAB-9982',
  processingPlantRegistrationNo: 'CDA/REG/2025/158',

  // Phytosanitary Certificate form
  distinguishingMarks: 'BWI-UK-LOT01',
  disinfestationTreatment: 'Fumigation with Methyl Bromide (CH3Br) at 48g/m³ for 24 hrs',

  // Customs Declaration form
  totalInvoiceValue: '1,250,000 LKR',
  totalPackages: 25,
  totalNetWeight: 1500.5,
};

// Alternative sample data sets for variety
export const SAMPLE_DATA_SETS: SampleDataMap[] = [
  SAMPLE_DATA_MAP,
  {
    // Alternative set 1
    consigneeName: 'Euro Organic Foods GmbH',
    consigneeAddress: 'Handelstraße 89\n10115 Berlin\nGermany',
    countryOfDestination: 'DE',
    productDescription: 'Organic Virgin Coconut Oil (Cold Pressed)',
    batchLotNumbers: 'VCO-2026-FEB-12',
    productionExpiryDates: 'Production: 2026-02-10, Expiry: 2028-02-10',
    microbiologicalTestReportId: 'ITI/2026/LAB-10234',
    processingPlantRegistrationNo: 'CDA/REG/2025/201',
    distinguishingMarks: 'EOF-DE-LOT02',
    disinfestationTreatment: 'Heat Treatment at 56°C for 30 minutes',
    totalInvoiceValue: '2,800,000 LKR',
    totalPackages: 40,
    totalNetWeight: 2400.0,
  },
  {
    // Alternative set 2
    consigneeName: 'Pacific Natural Trading Co.',
    consigneeAddress: '1250 Harbor Boulevard\nSan Francisco, CA 94107\nUnited States',
    countryOfDestination: 'US',
    productDescription: 'Organic Coconut Flour (Premium Grade)',
    batchLotNumbers: 'CF-2026-MAR-18',
    productionExpiryDates: 'Production: 2026-03-20, Expiry: 2027-03-20',
    microbiologicalTestReportId: 'ITI/2026/LAB-11056',
    processingPlantRegistrationNo: 'CDA/REG/2025/089',
    distinguishingMarks: 'PNT-US-LOT03',
    disinfestationTreatment: 'Phosphine (PH3) fumigation at 2g/m³ for 72 hrs',
    totalInvoiceValue: '950,000 LKR',
    totalPackages: 15,
    totalNetWeight: 850.25,
  },
];

// Get a sample value for a field name, with optional set index
export function getSampleValue(fieldName: string, setIndex: number = 0): unknown {
  const dataSet = SAMPLE_DATA_SETS[setIndex] || SAMPLE_DATA_MAP;
  return dataSet[fieldName];
}