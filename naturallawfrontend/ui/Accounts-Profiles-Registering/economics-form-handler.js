// Economics Form Handler
// This script handles form submission to the /api/v1/profile/economic endpoint

const API_BASE_URL = 'http://localhost:8080/api/v1';

// Get JWT token from localStorage
function getAuthToken() {
    return localStorage.getItem('authToken');
}

// Check if user is authenticated
function checkAuth() {
    const token = getAuthToken();
    if (!token) {
        alert('You must be logged in to update your economic profile. Redirecting to login page...');
        window.location.href = './Login.html';
        return false;
    }
    return true;
}

// Convert select value to numeric score (0-10 scale)
function convertSupportLevelToScore(selectValue) {
    const scoreMap = {
        'very-active': 10,
        'moderately-active': 8,
        'occasionally-active': 6,
        'cultural': 4,
        'studying': 2,
        'inactive': 0
    };
    return scoreMap[selectValue] || 0;
}

// Convert radio button value to numeric score
function convertYesNoToScore(value) {
    const scoreMap = {
        'yes': 10,
        'not-decided': 5,
        'indifferent': 3,
        'no': 0
    };
    return scoreMap[value] || 0;
}

// Parse comma-separated string into array
function parseCommaSeparatedToArray(value) {
    if (!value || value.trim() === '') return [];
    return value.split(',').map(item => item.trim()).filter(item => item !== '');
}

// Collect form data and prepare for API
function collectFormData() {
    const formData = {
        for_current_political_structure: convertYesNoToScore(
            document.querySelector('input[name="currentPoliticalStructure"]:checked')?.value
        ),
        for_capitalism: convertYesNoToScore(
            document.querySelector('input[name="capitalism"]:checked')?.value
        ),
        for_laws: convertYesNoToScore(
            document.querySelector('input[name="slavery"]:checked')?.value
        ),
        goods_services: parseCommaSeparatedToArray(
            document.getElementById('goodsServices')?.value
        ),
        affiliations: parseCommaSeparatedToArray(
            document.getElementById('affiliations')?.value
        ),
        support_of_alt_econ: convertSupportLevelToScore(
            document.getElementById('supportAltEcon')?.value
        ),
        support_alt_comm: convertSupportLevelToScore(
            document.getElementById('supportAltComm')?.value
        ),
        additional_text: document.getElementById('additionalInfo')?.value || ''
    };

    return formData;
}

// Create or update economic info
async function submitEconomicInfo(formData) {
    const token = getAuthToken();

    try {
        // First, try to check if economic info already exists
        const checkResponse = await fetch(`${API_BASE_URL}/profile/economic`, {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });

        let method = 'POST';
        let endpoint = `${API_BASE_URL}/profile/economic`;

        // If economic info exists (200), use PUT to update
        if (checkResponse.ok) {
            method = 'PUT';
        }
        // If not found (404), use POST to create
        // Any other error will be handled below

        const response = await fetch(endpoint, {
            method: method,
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to save economic information');
        }

        const data = await response.json();
        return { success: true, data };

    } catch (error) {
        console.error('Error submitting economic info:', error);
        return { success: false, error: error.message };
    }
}

// Handle form submission
async function handleFormSubmit(event) {
    event.preventDefault();

    if (!checkAuth()) {
        return;
    }

    // Show loading state
    const submitBtn = event.target.querySelector('button[type="submit"]');
    const originalText = submitBtn.textContent;
    submitBtn.textContent = 'Saving...';
    submitBtn.disabled = true;

    try {
        const formData = collectFormData();
        console.log('Submitting economic data:', formData);

        const result = await submitEconomicInfo(formData);

        if (result.success) {
            alert('Economic profile updated successfully!');
            console.log('Response data:', result.data);
        } else {
            alert(`Error: ${result.error}`);
        }
    } catch (error) {
        console.error('Form submission error:', error);
        alert('An unexpected error occurred. Please try again.');
    } finally {
        // Restore button state
        submitBtn.textContent = originalText;
        submitBtn.disabled = false;
    }
}

// Load existing economic info when page loads
async function loadExistingEconomicInfo() {
    const token = getAuthToken();
    if (!token) return;

    try {
        const response = await fetch(`${API_BASE_URL}/profile/economic`, {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });

        if (response.ok) {
            const data = await response.json();
            console.log('Loaded existing economic data:', data);
            populateFormWithData(data);
        } else if (response.status === 404) {
            console.log('No existing economic info found');
        } else {
            console.error('Error loading economic info:', response.status);
        }
    } catch (error) {
        console.error('Error loading existing economic info:', error);
    }
}

// Populate form with existing data
function populateFormWithData(data) {
    // Map numeric scores back to radio values
    const scoreToYesNo = (score) => {
        if (score >= 8) return 'yes';
        if (score >= 4) return 'not-decided';
        if (score >= 1) return 'indifferent';
        return 'no';
    };

    const scoreToSupportLevel = (score) => {
        if (score >= 9) return 'very-active';
        if (score >= 7) return 'moderately-active';
        if (score >= 5) return 'occasionally-active';
        if (score >= 3) return 'cultural';
        if (score >= 1) return 'studying';
        return 'inactive';
    };

    // Populate radio buttons
    if (data.for_current_political_structure !== undefined) {
        const value = scoreToYesNo(data.for_current_political_structure);
        const radio = document.querySelector(`input[name="currentPoliticalStructure"][value="${value}"]`);
        if (radio) radio.checked = true;
    }

    if (data.for_capitalism !== undefined) {
        const value = scoreToYesNo(data.for_capitalism);
        const radio = document.querySelector(`input[name="capitalism"][value="${value}"]`);
        if (radio) radio.checked = true;
    }

    if (data.for_laws !== undefined) {
        const value = scoreToYesNo(data.for_laws);
        const radio = document.querySelector(`input[name="slavery"][value="${value}"]`);
        if (radio) radio.checked = true;
    }

    // Populate arrays as comma-separated strings
    if (data.goods_services && Array.isArray(data.goods_services)) {
        const input = document.getElementById('goodsServices');
        if (input) input.value = data.goods_services.join(', ');
    }

    if (data.affiliations && Array.isArray(data.affiliations)) {
        const input = document.getElementById('affiliations');
        if (input) input.value = data.affiliations.join(', ');
    }

    // Populate select dropdowns
    if (data.support_of_alt_econ !== undefined) {
        const select = document.getElementById('supportAltEcon');
        if (select) select.value = scoreToSupportLevel(data.support_of_alt_econ);
    }

    if (data.support_alt_comm !== undefined) {
        const select = document.getElementById('supportAltComm');
        if (select) select.value = scoreToSupportLevel(data.support_alt_comm);
    }

    // Populate textarea
    if (data.additional_text) {
        const textarea = document.getElementById('additionalInfo');
        if (textarea) textarea.value = data.additional_text;
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    const form = document.querySelector('.registration-form');

    if (form) {
        form.addEventListener('submit', handleFormSubmit);
        console.log('Economic form handler initialized');

        // Load existing data if user is authenticated
        if (getAuthToken()) {
            loadExistingEconomicInfo();
        }
    } else {
        console.error('Registration form not found');
    }
});
