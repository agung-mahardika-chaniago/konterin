import { api } from '@/lib/api-client';
import type { ApiEnvelope } from '@/types/api';
import type {
  Claim,
  ClaimFilter,
  CreateClaimRequest,
  ReworkResult,
  UpdateClaimRequest,
  Warranty,
  WarrantyFilter,
} from './types';

function pageQuery(filter: object): string {
  const params = new URLSearchParams();
  Object.entries(filter).forEach(([key, value]) => {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      params.set(key, String(value));
    }
  });
  const query = params.toString();
  return query ? `?${query}` : '';
}

export async function listWarranties(filter: WarrantyFilter = {}): Promise<ApiEnvelope<Warranty[]>> {
  return api.getEnvelope<Warranty[]>(`/warranties${pageQuery(filter)}`);
}

export async function getWarranty(id: string): Promise<Warranty> {
  return api.get<Warranty>(`/warranties/${id}`);
}

export async function createClaim(warrantyId: string, payload: CreateClaimRequest): Promise<Claim> {
  return api.post<Claim>(`/warranties/${warrantyId}/claims`, payload);
}

export async function listClaims(filter: ClaimFilter = {}): Promise<ApiEnvelope<Claim[]>> {
  return api.getEnvelope<Claim[]>(`/claims${pageQuery(filter)}`);
}

export async function getClaim(id: string): Promise<Claim> {
  return api.get<Claim>(`/claims/${id}`);
}

export async function updateClaim(id: string, payload: UpdateClaimRequest): Promise<Claim> {
  return api.patch<Claim>(`/claims/${id}`, payload);
}

export async function createRework(claimId: string): Promise<ReworkResult> {
  return api.post<ReworkResult>(`/claims/${claimId}/rework`);
}

export function whatsappCustomerLink(phone: string, message: string): string {
  const normalized = phone.replace(/\D/g, '').replace(/^0/, '62');
  return `https://wa.me/${normalized}?text=${encodeURIComponent(message)}`;
}
