import React, { useState, useEffect } from 'react';

interface PartnerAttributes {
  chotot_id?: string;
  chotot_oid?: string;
  whatsapp_phone_number_id?: string;
}

interface FormData extends PartnerAttributes {
  whatsapp_system_token?: string;
}

interface PartnerAttributesFormProps {
  initialData: PartnerAttributes;
  onSave: (data: FormData) => Promise<void>;
  onCancel: () => void;
  onEdit?: () => void;
  isEditMode: boolean;
  isLoading: boolean;
  showSkipOption: boolean;
}

export const PartnerAttributesForm: React.FC<PartnerAttributesFormProps> = ({
  initialData,
  onSave,
  onCancel,
  onEdit,
  isEditMode,
  isLoading,
  showSkipOption
}) => {
  const [formData, setFormData] = useState<FormData>({
    chotot_id: '',
    chotot_oid: '',
    whatsapp_phone_number_id: '',
    whatsapp_system_token: ''
  });
  const [selectedPartners, setSelectedPartners] = useState<{
    chotot: boolean;
    whatsapp: boolean;
  }>({
    chotot: false,
    whatsapp: false
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (initialData) {
      setFormData({
        chotot_id: initialData.chotot_id || '',
        chotot_oid: initialData.chotot_oid || '',
        whatsapp_phone_number_id: initialData.whatsapp_phone_number_id || '',
        whatsapp_system_token: ''
      });

      // Set selected partners based on existing data
      setSelectedPartners({
        chotot: !!(initialData.chotot_id || initialData.chotot_oid),
        whatsapp: !!initialData.whatsapp_phone_number_id
      });
    }
  }, [initialData]);

  const handleInputChange = (field: keyof FormData, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  const handlePartnerToggle = (partner: 'chotot' | 'whatsapp') => {
    setSelectedPartners(prev => ({
      ...prev,
      [partner]: !prev[partner]
    }));

    // Clear form data for unselected partners
    if (partner === 'chotot' && selectedPartners.chotot) {
      setFormData(prev => ({
        ...prev,
        chotot_id: '',
        chotot_oid: ''
      }));
    }

    if (partner === 'whatsapp' && selectedPartners.whatsapp) {
      setFormData(prev => ({
        ...prev,
        whatsapp_phone_number_id: '',
        whatsapp_system_token: ''
      }));
    }
  };

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    // At least one partner must be selected
    if (!selectedPartners.chotot && !selectedPartners.whatsapp) {
      newErrors.general = 'Please select at least one partner to configure';
    }

    // Validate Chotot fields if selected
    if (selectedPartners.chotot) {
      if (!formData.chotot_id && !formData.chotot_oid) {
        newErrors.chotot = 'Please provide at least one Chotot field (ID or OID)';
      }
    }

    // Validate WhatsApp fields if selected
    if (selectedPartners.whatsapp) {
      if (!formData.whatsapp_phone_number_id || !formData.whatsapp_system_token) {
        newErrors.whatsapp = 'Both WhatsApp Phone Number ID and System Token are required';
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    try {
      // Only send data for selected partners
      const submitData: FormData = {};

      if (selectedPartners.chotot) {
        if (formData.chotot_id) submitData.chotot_id = formData.chotot_id;
        if (formData.chotot_oid) submitData.chotot_oid = formData.chotot_oid;
      }

      if (selectedPartners.whatsapp) {
        if (formData.whatsapp_phone_number_id) {
          submitData.whatsapp_phone_number_id = formData.whatsapp_phone_number_id;
        }
        if (formData.whatsapp_system_token) {
          submitData.whatsapp_system_token = formData.whatsapp_system_token;
        }
      }

      await onSave(submitData);
    } catch (error) {
      // Error is handled by parent component
    }
  };

  const hasExistingData = Object.values(initialData).some(value => value && value.trim() !== '');

  if (!isEditMode && hasExistingData) {
    // Read-only view for existing data
    return (
      <div className="space-y-4">
        {/* Chotot Section */}
        {(initialData.chotot_id || initialData.chotot_oid) && (
          <div className="border border-gray-200 rounded-lg p-4">
            <h3 className="font-medium text-gray-900 mb-3">Chotot Integration</h3>
            <div className="space-y-2">
              {initialData.chotot_id && (
                <div>
                  <label className="block text-sm font-medium text-gray-700">Chotot ID</label>
                  <p className="text-sm text-gray-900 bg-gray-50 px-3 py-2 rounded-md">
                    {initialData.chotot_id}
                  </p>
                </div>
              )}
              {initialData.chotot_oid && (
                <div>
                  <label className="block text-sm font-medium text-gray-700">Chotot OID</label>
                  <p className="text-sm text-gray-900 bg-gray-50 px-3 py-2 rounded-md">
                    {initialData.chotot_oid}
                  </p>
                </div>
              )}
            </div>
          </div>
        )}

        {/* WhatsApp Section */}
        {initialData.whatsapp_phone_number_id && (
          <div className="border border-gray-200 rounded-lg p-4">
            <h3 className="font-medium text-gray-900 mb-3">WhatsApp Integration</h3>
            <div>
              <label className="block text-sm font-medium text-gray-700">Phone Number ID</label>
              <p className="text-sm text-gray-900 bg-gray-50 px-3 py-2 rounded-md">
                {initialData.whatsapp_phone_number_id}
              </p>
            </div>
            <p className="text-xs text-gray-500 mt-2">
              System token is configured but hidden for security
            </p>
          </div>
        )}

        {/* Actions */}
        <div className="flex justify-end space-x-3 pt-4">
          <button
            onClick={onCancel}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
          >
            Close
          </button>
          {onEdit && (
            <button
              onClick={onEdit}
              className="flex items-center px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 transition-colors"
            >
              <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
              Edit
            </button>
          )}
        </div>
      </div>
    );
  }

  // Edit mode form
  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* General error */}
      {errors.general && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-md">
          <p className="text-sm text-red-800">{errors.general}</p>
        </div>
      )}

      {/* Partner Selection */}
      <div className="space-y-4">
        <h3 className="font-medium text-gray-900">Select Partners to Configure</h3>

        {/* Chotot */}
        <div className="relative">
          <label className="flex items-start p-3 border border-gray-200 rounded-lg hover:bg-gray-50 cursor-pointer transition-colors">
            <div className="flex items-center h-5">
              <input
                type="checkbox"
                checked={selectedPartners.chotot}
                onChange={() => handlePartnerToggle('chotot')}
                className="w-4 h-4 text-blue-600 bg-white border-gray-300 rounded focus:ring-blue-500 focus:ring-2"
              />
            </div>
            <div className="ml-3 flex-1">
              <div className="font-medium text-gray-900">Chotot Integration</div>
              <div className="text-sm text-gray-500">
                Connect with Chotot marketplace for product listings
              </div>
            </div>
          </label>
        </div>

        {/* WhatsApp */}
        <div className="relative">
          <label className="flex items-start p-3 border border-gray-200 rounded-lg hover:bg-gray-50 cursor-pointer transition-colors">
            <div className="flex items-center h-5">
              <input
                type="checkbox"
                checked={selectedPartners.whatsapp}
                onChange={() => handlePartnerToggle('whatsapp')}
                className="w-4 h-4 text-blue-600 bg-white border-gray-300 rounded focus:ring-blue-500 focus:ring-2"
              />
            </div>
            <div className="ml-3 flex-1">
              <div className="font-medium text-gray-900">WhatsApp Business Integration</div>
              <div className="text-sm text-gray-500">
                Enable WhatsApp messaging for customer communication
              </div>
            </div>
          </label>
        </div>
      </div>

      {/* Chotot Fields */}
      {selectedPartners.chotot && (
        <div className="border border-gray-200 rounded-lg p-4 space-y-3">
          <h4 className="font-medium text-gray-900">Chotot Configuration</h4>

          <div>
            <label htmlFor="chotot_id" className="block text-sm font-medium text-gray-700 mb-1">
              Chotot ID
            </label>
            <input
              type="text"
              id="chotot_id"
              value={formData.chotot_id}
              onChange={(e) => handleInputChange('chotot_id', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="Enter your Chotot ID"
            />
          </div>

          <div>
            <label htmlFor="chotot_oid" className="block text-sm font-medium text-gray-700 mb-1">
              Chotot OID
            </label>
            <input
              type="text"
              id="chotot_oid"
              value={formData.chotot_oid}
              onChange={(e) => handleInputChange('chotot_oid', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="Enter your Chotot OID"
            />
          </div>

          {errors.chotot && (
            <p className="text-sm text-red-600">{errors.chotot}</p>
          )}
        </div>
      )}

      {/* WhatsApp Fields */}
      {selectedPartners.whatsapp && (
        <div className="border border-gray-200 rounded-lg p-4 space-y-3">
          <h4 className="font-medium text-gray-900">WhatsApp Business Configuration</h4>

          <div>
            <label htmlFor="whatsapp_phone_number_id" className="block text-sm font-medium text-gray-700 mb-1">
              Phone Number ID <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              id="whatsapp_phone_number_id"
              value={formData.whatsapp_phone_number_id}
              onChange={(e) => handleInputChange('whatsapp_phone_number_id', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="Enter Phone Number ID"
              required
            />
          </div>

          <div>
            <label htmlFor="whatsapp_system_token" className="block text-sm font-medium text-gray-700 mb-1">
              System Token <span className="text-red-500">*</span>
            </label>
            <input
              type="password"
              id="whatsapp_system_token"
              value={formData.whatsapp_system_token}
              onChange={(e) => handleInputChange('whatsapp_system_token', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="Enter System Token"
              required
            />
            <p className="text-xs text-gray-500 mt-1">
              This token will be encrypted and stored securely
            </p>
          </div>

          {errors.whatsapp && (
            <p className="text-sm text-red-600">{errors.whatsapp}</p>
          )}
        </div>
      )}

      {/* Actions */}
      <div className="flex justify-end space-x-3 pt-4">
        {showSkipOption && (
          <button
            type="button"
            onClick={onCancel}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
          >
            Skip for now
          </button>
        )}

        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
        >
          <svg className="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
          Cancel
        </button>

        <button
          type="submit"
          disabled={isLoading}
          className="flex items-center px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {isLoading ? (
            <>
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
              Saving...
            </>
          ) : (
            <>
              <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              Save
            </>
          )}
        </button>
      </div>
    </form>
  );
};
