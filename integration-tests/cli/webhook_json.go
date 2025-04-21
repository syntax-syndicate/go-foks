// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cli

const paymentSuccessWebhookJson = `
{
  "id": "{{ .EventID }}",
  "object": "event",
  "api_version": "2025-02-24.acacia",
  "created": {{ .TimeCreated }}, 
  "data": {
    "object": {
      "id": "{{ .InvoiceID }}",
      "object": "invoice",
      "account_country": "US",
      "account_name": "Sandbox",
      "account_tax_ids": null,
      "amount_due": 95,
      "amount_paid": 95,
      "amount_remaining": 0,
      "amount_shipping": 0,
      "application": null,
      "application_fee_amount": null,
      "attempt_count": 1,
      "attempted": true,
      "auto_advance": false,
      "automatic_tax": {
        "enabled": false,
        "liability": null,
        "status": null
      },
      "automatically_finalizes_at": null,
      "billing_reason": "subscription_cycle",
      "charge": "{{ .ChargeID }}",
      "collection_method": "charge_automatically",
      "created": {{ .TimeCreated }},
      "currency": "usd",
      "custom_fields": null,
      "customer": "{{ .CustomerID }}",
      "customer_address": null,
      "customer_email": "{{ .Email }}",
      "customer_name": null,
      "customer_phone": null,
      "customer_shipping": null,
      "customer_tax_exempt": "none",
      "customer_tax_ids": [],
      "default_payment_method": null,
      "default_source": null,
      "default_tax_rates": [],
      "description": null,
      "discount": null,
      "discounts": [],
      "due_date": null,
      "effective_at": 1727228884,
      "ending_balance": 0,
      "footer": null,
      "from_invoice": null,
      "hosted_invoice_url": "https://invoice.stripe.com/i/acct_1PykJVP3NyTLHrvd/test_YWNjdF8xUHlrSlZQM055VExIcnZkLF9RdVpURk5EM1RHM0Uxck9aZkQwWk0xNUxjdVhPS0tvLDExNzc2OTY4Nw0200aoSXoSAo?s=ap",
      "invoice_pdf": "https://pay.stripe.com/invoice/acct_1PykJVP3NyTLHrvd/test_YWNjdF8xUHlrSlZQM055VExIcnZkLF9RdVpURk5EM1RHM0Uxck9aZkQwWk0xNUxjdVhPS0tvLDExNzc2OTY4Nw0200aoSXoSAo/pdf?s=ap",
      "issuer": {
        "type": "self"
      },
      "last_finalization_error": null,
      "latest_revision": null,
      "lines": {
        "object": "list",
        "data": [
          {
            "id": "{{ .IlID}}",
            "object": "line_item",
            "amount": {{ .Amount }},
            "amount_excluding_tax": {{ .Amount }},
            "currency": "usd",
            "description": "1 × Dingo 2 Eta (at $0.95 / day)",
            "discount_amounts": [],
            "discountable": true,
            "discounts": [],
            "invoice": "{{ .InvoiceID }}",
            "livemode": false,
            "metadata": {},
            "period": {
              "end": {{ .PeriodEnd }},
              "start": {{ .PeriodStart }}
            },
            "plan": {
              "id": "{{ .PriceID }}",
              "object": "plan",
              "active": true,
              "aggregate_usage": null,
              "amount": 95,
              "amount_decimal": "95",
              "billing_scheme": "per_unit",
              "created": {{ .TimeCreated }},
              "currency": "usd",
              "interval": "day",
              "interval_count": 1,
              "livemode": false,
              "metadata": {},
              "meter": null,
              "nickname": null,
              "product": "{{ .PlanID }}",
              "tiers_mode": null,
              "transform_usage": null,
              "trial_period_days": null,
              "usage_type": "licensed"
            },
            "price": {
              "id": "{{ .PriceID }}",
              "object": "price",
              "active": true,
              "billing_scheme": "per_unit",
              "created": {{ .TimeCreated }},
              "currency": "usd",
              "custom_unit_amount": null,
              "livemode": false,
              "lookup_key": null,
              "metadata": {},
              "nickname": null,
              "product": "{{ .PlanID }}", 
              "recurring": {
                "aggregate_usage": null,
                "interval": "day",
                "interval_count": 1,
                "meter": null,
                "trial_period_days": null,
                "usage_type": "licensed"
              },
              "tax_behavior": "unspecified",
              "tiers_mode": null,
              "transform_quantity": null,
              "type": "recurring",
              "unit_amount": {{ .Amount }},
              "unit_amount_decimal": "{{ .AmountDecimal }}"
            },
            "proration": false,
            "proration_details": {
              "credited_items": null
            },
            "quantity": 1,
            "subscription": "{{ .SubscriptionID }}",
            "subscription_item": "{{ .SubscriptionItemID }}",
            "tax_amounts": [],
            "tax_rates": [],
            "type": "subscription",
            "unit_amount_excluding_tax": "95"
          }
        ],
        "has_more": false,
        "total_count": 1,
        "url": "/v1/invoices/{{ .InvoiceID }}/lines"
      },
      "livemode": false,
      "metadata": {},
      "next_payment_attempt": null,
      "number": "C0381EFB-0002",
      "on_behalf_of": null,
      "paid": true,
      "paid_out_of_band": false,
      "payment_intent": "{{ .PaymentIntentID }}",
      "payment_settings": {
        "default_mandate": null,
        "payment_method_options": {
          "acss_debit": null,
          "bancontact": null,
          "card": {
            "request_three_d_secure": "automatic"
          },
          "customer_balance": null,
          "konbini": null,
          "sepa_debit": null,
          "us_bank_account": null
        },
        "payment_method_types": null
      },
      "period_end": {{ .TimeCreated }}, 
      "period_start": {{ .TimeCreated }},
      "post_payment_credit_notes_amount": 0,
      "pre_payment_credit_notes_amount": 0,
      "quote": null,
      "receipt_number": null,
      "rendering": null,
      "shipping_cost": null,
      "shipping_details": null,
      "starting_balance": 0,
      "statement_descriptor": null,
      "status": "paid",
      "status_transitions": {
        "finalized_at": {{ .TimeCreated }},
        "marked_uncollectible_at": null,
        "paid_at": {{ .TimeCreated }},
        "voided_at": null
      },
      "subscription": "{{ .SubscriptionID }}",
      "subscription_details": {
        "metadata": {}
      },
      "subtotal": {{ .Amount }},
      "subtotal_excluding_tax": {{ .Amount }},
      "tax": null,
      "test_clock": null,
      "total": 95,
      "total_discount_amounts": [],
      "total_excluding_tax": 95,
      "total_tax_amounts": [],
      "transfer_data": null,
      "webhooks_delivered_at": null
    },
    "previous_attributes": null
  },
  "livemode": false,
  "pending_webhooks": 0,
  "request": {
    "id": "{{ .RequestID }}",
    "idempotency_key": "{{ .IdempotencyKey }}"
  },
  "type": "invoice.payment_succeeded"
}`
