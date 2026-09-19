# Frontend & UI Design Guidelines

> Guidelines for building responsive, accessible, distinctive, and production-quality interfaces with HTMX.

This document defines the frontend and visual design standards for this project.

The main goals are:

1. Clear visual identity
2. Excellent usability
3. Responsive behavior
4. Accessibility
5. Semantic HTML
6. Minimal JavaScript
7. Fast perceived performance
8. Consistent interaction patterns

The interface must never look like a generic generated dashboard.

Every product must have its own visual identity.

---

# 1. Core Design Principle

The interface must communicate what the product is before the user reads extensive text.

Visual design must derive from the product domain.

Do not build interfaces by combining generic:

* cards
* rounded rectangles
* gradients
* random icons
* oversized headings
* dashboard grids
* glassmorphism
* generic SaaS layouts

without a clear visual reason.

A component must exist because it solves a UX problem, not because it is fashionable.

---

# 2. Product Identity

Before implementing the interface, define:

```text
Product purpose
Target audience
Primary actions
Visual personality
Typography
Color palette
Spacing rhythm
Border style
Radius style
Icon language
Illustration style
Motion language
```

The design must remain consistent with these decisions.

---

# 3. Identity Must Come From the Domain

Visual identity should reference the application's purpose.

For a nursing-related application, for example, potential visual references may include:

```text
clinical organization
healthcare signage
medical records
hospital identification
scrubs
medical cross geometry
heartbeat rhythm
medical charts
care and human assistance
```

Do not translate these ideas literally everywhere.

Use them subtly through:

* shapes
* spacing
* iconography
* color
* typography
* interaction details

The result should feel related to healthcare without looking like a generic hospital template.

---

# 4. Avoid Generic AI Design

Do not automatically produce:

```text
purple-to-blue gradients
giant hero headings
floating glass cards
excessive rounded corners
random pill badges
dozens of cards
decorative blobs
identical dashboard widgets
gradient buttons
gratuitous animations
```

These patterns may be used only when justified by the product identity.

---

# 5. Establish a Design Direction First

Before creating pages, define a visual direction.

Example:

```text
Personality:
Clean, professional, trustworthy, calm.

Visual references:
Hospital signage + modern job board.

Geometry:
Mostly rectangular with subtle radius.

Typography:
Readable sans-serif with strong hierarchy.

Color:
Neutral surfaces with one healthcare-related accent color.

Density:
Moderate information density.

Iconography:
Simple outlined medical and navigation icons.
```

All subsequent components should follow this direction.

---

# 6. Design Tokens

Centralize visual decisions using CSS custom properties.

Example:

```css
:root {
	--color-background: #f7f9f8;
	--color-surface: #ffffff;
	--color-text: #17211d;
	--color-text-muted: #66736d;

	--color-primary: #167c67;
	--color-primary-hover: #126b59;

	--color-border: #dce4e0;
	--color-danger: #b42318;

	--radius-sm: 0.375rem;
	--radius-md: 0.625rem;
	--radius-lg: 0.875rem;

	--spacing-sm: 0.5rem;
	--spacing-md: 1rem;
	--spacing-lg: 1.5rem;
```
