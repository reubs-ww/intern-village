---
name: mobile-frontend-design
description: Create distinctive, production-grade mobile interfaces with exceptional design quality. Use this skill when the user asks to build mobile apps, mobile components, or mobile-first interfaces (examples include React Native apps, Flutter apps, iOS/Android screens, mobile web apps, PWAs, or mobile-responsive designs). Generates creative, polished mobile UI that avoids generic patterns and prioritizes touch-first, gesture-driven experiences.
---

This skill guides creation of distinctive, production-grade mobile interfaces that feel native and delightful. Implement real working code with exceptional attention to mobile-specific aesthetics, gestures, and platform conventions.

The user provides mobile frontend requirements: a screen, component, app, or interface to build. They may include context about the platform (iOS, Android, cross-platform), framework (React Native, Flutter, SwiftUI, Jetpack Compose), or technical constraints.

## Design Thinking

Before coding, understand the context and commit to a BOLD aesthetic direction:
- **Platform**: iOS (Human Interface Guidelines), Android (Material Design 3), or cross-platform? Each has distinct personality.
- **Purpose**: What problem does this solve? What's the core user action?
- **Tone**: Pick an extreme: sleek/premium, playful/bouncy, brutalist/raw, organic/fluid, editorial/content-first, glassmorphic/layered, neomorphic/soft, bold/high-contrast, minimal/zen, maximalist/immersive.
- **Thumb Zone**: Where do primary actions live? Bottom navigation, floating actions, gesture-based?
- **Differentiation**: What micro-interaction or visual detail makes this MEMORABLE?

**CRITICAL**: Mobile is intimate. Users hold it in their hands. Every pixel, every transition, every haptic moment matters more than on desktop.

Then implement working code (React Native, Flutter, SwiftUI, etc.) that is:
- Touch-optimized with generous tap targets (44pt minimum)
- Gesture-aware with swipe, pull, and long-press interactions
- Performant at 60fps with smooth animations
- Platform-appropriate while maintaining unique character

## Mobile Aesthetics Guidelines

### Touch & Gesture Design
- **Tap targets**: Minimum 44x44pt. Generous padding. Visible feedback states (pressed, disabled).
- **Gestures**: Swipe-to-dismiss, pull-to-refresh, long-press menus, pan gestures. Make them discoverable.
- **Haptics**: Specify haptic feedback types (light, medium, heavy, selection, success, error).
- **Edge gestures**: Respect system gestures (back swipe on iOS, edge swipe on Android).

### Typography
- **System fonts** can work beautifully on mobile (SF Pro, Roboto) - use weight and size contrast.
- **Custom fonts**: Choose distinctive display fonts for headers. Ensure readability at small sizes.
- **Dynamic type**: Support accessibility text scaling. Test at largest sizes.
- **Hierarchy**: Mobile screens are small. Make hierarchy OBVIOUS through size, weight, and spacing.

### Color & Theme
- **Dark mode**: Design for both light and dark. Not an afterthought.
- **OLED considerations**: True blacks (#000) save battery and look stunning on OLED.
- **Vibrancy**: Use system blur and vibrancy effects (iOS) or elevation (Android).
- **Semantic colors**: Use platform semantic colors for adaptability.

### Motion & Animation
- **Spring physics**: Use spring-based animations for natural feel. Avoid linear easing.
- **Shared element transitions**: Connect screens with morphing elements.
- **Micro-interactions**: Button presses, toggle switches, loading states. Every state change is an opportunity.
- **Performance**: Animate transform and opacity only. Use native drivers. Target 60fps.
- **Reduce motion**: Respect accessibility settings. Provide reduced motion alternatives.

### Spatial Composition
- **Safe areas**: Respect notches, dynamic islands, home indicators.
- **Bottom-focused**: Primary actions near thumb reach. Top for status, bottom for action.
- **Cards and surfaces**: Use elevation and shadow to create depth hierarchy.
- **Negative space**: Mobile needs breathing room. Don't cram.
- **Full-bleed media**: Images and videos should feel immersive.

### Platform-Specific Patterns

**iOS (Human Interface Guidelines)**:
- Large titles that collapse on scroll
- SF Symbols for iconography
- Sheet presentations (half-sheets, full-sheets)
- Haptic feedback on interactions
- Glassmorphism and vibrancy

**Android (Material Design 3)**:
- Dynamic color theming from wallpaper
- FAB placement and behavior
- Bottom sheets and dialogs
- Predictive back gestures
- Elevation and tonal surfaces

### Anti-Patterns to Avoid
- Generic hamburger menus (use bottom tabs or contextual navigation)
- Tiny touch targets
- Desktop-style hover effects
- Ignoring platform conventions entirely
- Static, lifeless interfaces without motion
- Cookie-cutter templates without character

## Framework-Specific Notes

**React Native**: Use Reanimated for 60fps animations. Use Gesture Handler for touch. Prefer native components.

**Flutter**: Use Cupertino widgets for iOS feel, Material 3 for Android. Use Hero for shared transitions.

**SwiftUI**: Embrace declarative animations, matchedGeometryEffect, sensory feedback.

**Jetpack Compose**: Use Material 3 components, animate*AsState, shared element transitions.

**Mobile Web/PWA**: Use CSS touch-action, viewport units (dvh), overscroll-behavior. Consider installing as PWA.

Remember: Mobile is where design craft matters most. The interface lives in the user's pocket, in their hands. Make every interaction feel intentional, every animation feel alive, every screen feel like it belongs on this platform.
