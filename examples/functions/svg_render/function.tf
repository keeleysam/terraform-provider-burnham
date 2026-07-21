# Render an SVG to a PNG (returned as base64) at 240px wide. Deterministic: the
# same SVG always produces the same bytes, so the result is safe to embed in a
# data URI, write to a file, or push to an API without churning the plan.
locals {
  badge = <<-EOT
    <svg xmlns="http://www.w3.org/2000/svg" width="120" height="40" viewBox="0 0 120 40">
      <defs>
        <linearGradient id="g" x1="0" y1="0" x2="1" y2="0">
          <stop offset="0" stop-color="#2563eb" />
          <stop offset="1" stop-color="#7c3aed" />
        </linearGradient>
      </defs>
      <rect width="120" height="40" rx="6" fill="url(#g)" />
      <circle cx="20" cy="20" r="10" fill="#f59e0b" />
    </svg>
  EOT
}

output "png_base64" {
  value = provider::burnham::svg_render(local.badge, { width = 240 })
}
