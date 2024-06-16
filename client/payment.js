document.addEventListener("DOMContentLoaded", async function () {
  const { publishableKey } = await fetch("/config").then((r) => r.json());
  const stripe = Stripe(publishableKey);

  const { clientSecret } = await fetch("/create-payment-intent", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      currency: "usd",
      paymentMethodType: "card",
    }),
  }).then((r) => r.json());

  const appearance = {
    theme: "stripe",
    variables: {
      // Colors
      colorPrimary: "#528bff", // Primary color for actions
      colorBackground: "#081411", // Dark background color
      colorText: "#E8F9F7", // Light text color
      colorDanger: "#ff005e", // Danger color for errors
      colorPrimaryText: "#E8F9F7", // Primary text color
      colorSecondaryText: "#828997", // Muted text color
      colorPlaceholder: "#9ca3af", // Placeholder text color
      colorIcon: "#528bff", // Icon color
      colorIconAccent: "#528bff", // Icon accent color
      colorIconDisabled: "#4b5263", // Disabled icon color

      // Font and spacing
      fontFamily: "Ideal Sans, system-ui, sans-serif",
      spacingUnit: "4px", // Increased spacing unit for larger input padding
      borderRadius: "6px", // Slightly larger border radius for rounded corners

      // Input and form control
      colorInputText: "#E8F9F7", // Input text color
      colorInputBackground: "#081411", // Input background color
      colorInputBorder: "#528bff", // Input border color
      colorInputPlaceholder: "#9ca3af", // Input placeholder color
      colorInputDisabled: "#4b5263", // Input disabled color

      // Control colors
      colorControlBackground: "#124946", // Control background color
      colorControlBorder: "#528bff", // Control border color
      colorControlHovered: "#31353b", // Control hover color
      colorControlActive: "#31353b", // Control active color
      colorControlDisabled: "#42464d", // Control disabled color

      // Status colors
      colorDangerText: "#ff005e", // Error text color
      colorSuccessText: "#4a946e", // Success text color

      // Checkbox
      colorCheckbox: "#528bff", // Checkbox color
      colorCheckboxHover: "#528bff", // Checkbox hover color
    },
    rules: {
      ".Input": {
        color: "var(--colorInputText);",
        backgroundColor: "var(--colorInputBackground);",
        borderColor: "var(--colorInputBorder);",
        padding: "18px 20px;", // Increased padding for larger input fields
        fontSize: "16px;", // Increased font size for readability
        "::placeholder": "color: var(--colorInputPlaceholder);", // Properly formatted CSS string
        ":disabled": "color: var(--colorInputDisabled);",
      },
      ".Label": {
        color: "var(--colorPrimaryText);",
        fontSize: "14px;", // Slightly larger font size for labels
      },
      ".Tab, .Button": {
        color: "var(--colorPrimaryText);",
        backgroundColor: "var(--colorBackground);",
        borderColor: "var(--colorPrimary);",
        padding: "14px 22px;", // Increased padding for buttons
        fontSize: "16px;", // Consistent font size with inputs
        borderRadius: "19px;", // Consistent border radius with inputs
        ":hover": "background-color: var(--colorControlHovered);",
        ":active": "background-color: var(--colorControlActive);",
        ":disabled":
          "background-color: var(--colorControlDisabled); color: var(--colorSecondaryText);",
      },
      ".Card": {
        backgroundColor: "var(--colorBackground);",
        borderColor: "var(--colorPrimary);",
        color: "var(--colorPrimaryText);",
      },
      ".Checkbox": {
        color: "var(--colorCheckbox);",
        ":hover": "color: var(--colorCheckboxHover);",
      },
      ".Error": {
        color: "var(--colorDangerText);",
        fontSize: "14px;", // Increased font size for better visibility
      },
      ".Success": {
        color: "var(--colorSuccessText);",
        fontSize: "14px;", // Increased font size for better visibility
      },
    },
  };

  // Pass the appearance object to the Elements instance
  const elements = stripe.elements({ clientSecret, appearance });

  const paymentElement = elements.create("payment");
  paymentElement.mount("#payment-element");

  const form = document.getElementById("payment-form");
  form.addEventListener("submit", async function (e) {
    e.preventDefault();

    const { error } = await stripe.confirmPayment({
      elements,
      confirmParams: {
        return_url: window.location.href.split("?")[0] + "complete.html",
      },
    });

    if (error) {
      const messages = document.getElementById("error-messages");
      messages.innerText = error.message;
    }
  });
});
