import React, { useState, useEffect } from 'react';
import { PartnerAttributesForm } from './PartnerAttributesForm';
import { api } from '../utils/api';

interface PartnerAttributes {
  chotot_id?: string;
  chotot_oid?: string;
  whatsapp_phone_number_id?: string;
}

interface PartnerAttributesModalProps {
  isOpen: boolean;
  onClose: () => void;
  showSkipOption?: boolean;
  title?: string;
}

export const PartnerAttributesModal: React.FC<PartnerAttributesModalProps> = ({
  isOpen,
  onClose,
  showSkipOption = false,
  title = 'Partner Integration Settings'
}) => {
  const [attributes, setAttributes] = useState<PartnerAttributes>({});
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string>('');
  const [isEditMode, setIsEditMode] = useState(false);

  useEffect(() => {
    if (isOpen) {
      fetchAttributes();
    }
  }, [isOpen]);

  const fetchAttributes = async () => {
    try {
      setIsLoading(true);
      setError('');
      const response = await api.getPartnerAttributes();
      setAttributes(response);

      // Determine if we should start in edit mode
      const hasAnyAttribute = Object.values(response).some(value => value && value.trim() !== '');
      setIsEditMode(!hasAnyAttribute);
    } catch (err) {
      console.error('Failed to fetch partner attributes:', err);
      setError('Failed to load partner settings');
      setIsEditMode(true); // Start in edit mode if fetch fails
    } finally {
      setIsLoading(false);
    }
  };

  const handleSave = async (formData: PartnerAttributes & { whatsapp_system_token?: string }) => {
    try {
      setIsLoading(true);
      setError('');

      await api.updatePartnerAttributes(formData);

      // Refresh the data after successful update
      await fetchAttributes();
      setIsEditMode(false);
    } catch (err) {
      console.error('Failed to update partner attributes:', err);
      setError('Failed to save partner settings. Please try again.');
      throw err; // Re-throw to let form handle the error
    } finally {
      setIsLoading(false);
    }
  };

  const handleEdit = () => {
    setIsEditMode(true);
    setError('');
  };

  const handleCancel = () => {
    if (showSkipOption) {
      // If this is the initial setup popup, close the modal
      onClose();
    } else {
      // If this is from settings, just exit edit mode
      setIsEditMode(false);
      setError('');
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-white rounded-lg p-6 w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-gray-900">{title}</h2>
          <button
            onClick={onClose}
            className="p-1 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-full transition-colors"
            aria-label="Close modal"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Error message */}
        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md">
            <p className="text-sm text-red-800">{error}</p>
          </div>
        )}

        {/* Loading state */}
        {isLoading && !isEditMode ? (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            <span className="ml-3 text-gray-600">Loading...</span>
          </div>
        ) : (
          <>
            {/* Form */}
            <PartnerAttributesForm
              initialData={attributes}
              onSave={handleSave}
              onCancel={handleCancel}
              isEditMode={isEditMode}
              isLoading={isLoading}
              showSkipOption={showSkipOption}
              onEdit={handleEdit}
            />
          </>
        )}
      </div>
    </div>
  );
};
