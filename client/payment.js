document.addEventListener("DOMContentLoaded", async function () {
  const { publishableKey } = await fetch("/config").then((r) => r.json());
  const stripe = Stripe(publishableKey);

  let paymentIntentID; // Store the payment intent ID
  let clientSecret;
  let elements;

  const createOrUpdatePaymentIntent = async (amount) => {
    const response = await fetch("/create-payment-intent", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        currency: "usd",
        amount: amount * 100, // Convert to cents
        paymentMethodType: "card",
      }),
    });

    const data = await response.json();
    clientSecret = data.clientSecret;
    paymentIntentID = data.paymentIntentID;

    if (!elements) {
      const appearance = {
        theme: "stripe",
        variables: {
          // Colors and other variables
          colorPrimary: "#528bff",
          colorBackground: "#081411",
          colorText: "#E8F9F7",
          colorDanger: "#ff005e",
          colorPrimaryText: "#E8F9F7",
          colorSecondaryText: "#828997",
          colorPlaceholder: "#9ca3af",
          colorIcon: "#528bff",
          colorIconAccent: "#528bff",
          colorIconDisabled: "#4b5263",
          fontFamily: "Ideal Sans, system-ui, sans-serif",
          spacingUnit: "4px",
          borderRadius: "6px",
          colorInputText: "#E8F9F7",
          colorInputBackground: "#081411",
          colorInputBorder: "#528bff",
          colorInputPlaceholder: "#9ca3af",
          colorInputDisabled: "#4b5263",
          colorControlBackground: "#124946",
          colorControlBorder: "#528bff",
          colorControlHovered: "#31353b",
          colorControlActive: "#31353b",
          colorControlDisabled: "#42464d",
          colorDangerText: "#ff005e",
          colorSuccessText: "#4a946e",
          colorCheckbox: "#528bff",
          colorCheckboxHover: "#528bff",
        },
        rules: {
          ".Input": {
            color: "var(--colorInputText);",
            backgroundColor: "var(--colorInputBackground);",
            borderColor: "var(--colorInputBorder);",
            padding: "18px 20px;",
            fontSize: "16px;",
            "::placeholder": "color: var(--colorInputPlaceholder);",
            ":disabled": "color: var(--colorInputDisabled);",
          },
          ".Label": {
            color: "var(--colorPrimaryText);",
            fontSize: "14px;",
          },
          ".Tab, .Button": {
            color: "var(--colorPrimaryText);",
            backgroundColor: "var(--colorBackground);",
            borderColor: "var(--colorPrimary);",
            padding: "14px 22px;",
            fontSize: "16px;",
            borderRadius: "19px;",
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
            fontSize: "14px;",
          },
          ".Success": {
            color: "var(--colorSuccessText);",
            fontSize: "14px;",
          },
        },
      };

      elements = stripe.elements({ clientSecret, appearance });
      const paymentElement = elements.create("payment");
      paymentElement.mount("#payment-element");
    } else {
      elements.update({ clientSecret });
    }
  };

  await createOrUpdatePaymentIntent(1);

  const radioButtons = document.querySelectorAll(".sr-radio-input");
  radioButtons.forEach((radio) => {
    radio.addEventListener("change", async function () {
      const amount =
        this.value === "custom"
          ? document.getElementById("custom-amount-input").value
          : this.value;
      await createOrUpdatePaymentIntent(amount);
    });
  });

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
