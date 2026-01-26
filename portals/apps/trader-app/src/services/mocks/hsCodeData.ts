import type { HSCode } from '../types/hsCode'

export const mockHSCodes: HSCode[] = [
  // Chapter 09 - Coffee, tea, mate and spices
  { id: '1', hsCode: '09', description: 'Coffee, tea, mate and spices', category: 'Chapter 09', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },

  // 09.02 - Tea, whether or not flavoured
  { id: '2', hsCode: '0902', description: 'Tea, whether or not flavoured', category: 'Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },

  // 0902.10 - Green tea (not fermented) in immediate packings not exceeding 3 kg
  { id: '3', hsCode: '090210', description: 'Green tea (not fermented) in immediate packings of a content not exceeding 3 kg', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '4', hsCode: '09021011', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (≤4g packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '5', hsCode: '09021012', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (≤4g packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '6', hsCode: '09021013', description: 'Other, flavoured (≤4g packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '7', hsCode: '09021019', description: 'Other (≤4g packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '8', hsCode: '09021021', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (4g-1kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '9', hsCode: '09021022', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (4g-1kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '10', hsCode: '09021023', description: 'Other, flavoured (4g-1kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '11', hsCode: '09021029', description: 'Other (4g-1kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '12', hsCode: '09021031', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (1kg-3kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '13', hsCode: '09021032', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (1kg-3kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '14', hsCode: '09021033', description: 'Other, flavoured (1kg-3kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '15', hsCode: '09021039', description: 'Other (1kg-3kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },

  // 0902.20 - Other green tea (not fermented)
  { id: '16', hsCode: '090220', description: 'Other green tea (not fermented)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '17', hsCode: '09022011', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (3kg-5kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '18', hsCode: '09022012', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (3kg-5kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '19', hsCode: '09022013', description: 'Other, flavoured (3kg-5kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '20', hsCode: '09022019', description: 'Other (3kg-5kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '21', hsCode: '09022021', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (5kg-10kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '22', hsCode: '09022022', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (5kg-10kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '23', hsCode: '09022023', description: 'Other, flavoured (5kg-10kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '24', hsCode: '09022029', description: 'Other (5kg-10kg packing)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '25', hsCode: '09022091', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (Other)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '26', hsCode: '09022092', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (Other)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '27', hsCode: '09022093', description: 'Other, flavoured (Other)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '28', hsCode: '09022099', description: 'Other (Other)', category: 'Green Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },

  // 0902.30 - Black tea (fermented) and partly fermented tea, in immediate packings not exceeding 3 kg
  { id: '29', hsCode: '090230', description: 'Black tea (fermented) and partly fermented tea, in immediate packings of a content not exceeding 3 kg', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '30', hsCode: '09023011', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (≤4g packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '31', hsCode: '09023012', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (≤4g packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '32', hsCode: '09023013', description: 'Other, Flavoured (≤4g packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '33', hsCode: '09023019', description: 'Other (≤4g packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '34', hsCode: '09023021', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (4g-1kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '35', hsCode: '09023022', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (4g-1kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '36', hsCode: '09023023', description: 'Other, Flavoured (4g-1kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '37', hsCode: '09023029', description: 'Other (4g-1kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '38', hsCode: '09023031', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (1kg-3kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '39', hsCode: '09023032', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (1kg-3kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '40', hsCode: '09023033', description: 'Other, Flavoured (1kg-3kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '41', hsCode: '09023039', description: 'Other (1kg-3kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },

  // 0902.40 - Other black tea (fermented) and other partly fermented tea
  { id: '42', hsCode: '090240', description: 'Other black tea (fermented) and other partly fermented tea', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '43', hsCode: '09024011', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (3kg-5kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '44', hsCode: '09024012', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (3kg-5kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '45', hsCode: '09024013', description: 'Other, flavoured (3kg-5kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '46', hsCode: '09024019', description: 'Other (3kg-5kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '47', hsCode: '09024021', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (5kg-10kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '48', hsCode: '09024022', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (5kg-10kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '49', hsCode: '09024023', description: 'Other, flavoured (5kg-10kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '50', hsCode: '09024029', description: 'Other (5kg-10kg packing)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '51', hsCode: '09024091', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, flavoured (Other)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '52', hsCode: '09024092', description: 'Certified by Sri Lanka Tea Board as wholly of Sri Lanka origin, Other (Other)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '53', hsCode: '09024093', description: 'Other, flavoured (Other)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
  { id: '54', hsCode: '09024099', description: 'Other (Other)', category: 'Black Tea', createdAt: '2025-01-01T00:00:00Z', updatedAt: '2025-01-01T00:00:00Z' },
]