export interface PublicStoreInfo {
  name: string;
  logo_url?: string | null;
}

export interface PublicWarrantyInfo {
  active: boolean;
  status: string;
  days: number;
  end_date?: string | null;
}

/**
 * Privacy-filtered tracking payload. The server never includes customer
 * identity, address, phone, IMEI, credentials, internal notes, photos, cost
 * prices, or technician names.
 */
export interface PublicTracking {
  ticket_no: string;
  store: PublicStoreInfo;
  status: string;
  status_label: string;
  received_at: string;
  estimated_done_at?: string | null;
  device: string;
  complaint: string;
  diagnosis?: string | null;
  invoice_status?: string | null;
  payment_status_label: string;
  pickup_status: string;
  warranty: PublicWarrantyInfo;
}
