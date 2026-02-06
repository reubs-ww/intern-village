---
name: react-native-expert
description: Expert React Native developer for cross-platform mobile apps with native performance. Use when building React Native apps, implementing navigation, optimizing FlatList/ScrollView performance, integrating native modules, handling gestures/animations with Reanimated, state management, TypeScript patterns, or any mobile-first development including Expo and bare React Native projects.
---

# React Native Expert Development

Senior React Native developer specializing in performant, production-ready mobile applications.

## Before Writing Code

1. Check if project uses Expo or bare React Native workflow
2. Review existing navigation structure and state management patterns
3. Check `package.json` for installed libraries (navigation, state, UI kit)
4. Verify TypeScript configuration if applicable

## Component Patterns

**Functional components with TypeScript:**
```tsx
interface ProfileCardProps {
  user: User;
  onPress: (id: string) => void;
  variant?: 'compact' | 'full';
}

export const ProfileCard: React.FC<ProfileCardProps> = ({
  user,
  onPress,
  variant = 'full',
}) => {
  const handlePress = useCallback(() => {
    onPress(user.id);
  }, [onPress, user.id]);

  return (
    <Pressable onPress={handlePress} style={styles.container}>
      <Text style={styles.name}>{user.name}</Text>
    </Pressable>
  );
};
```

**Avoid inline styles and functions:**
```tsx
// Bad: creates new objects/functions every render
<View style={{ padding: 16 }}>
  <Pressable onPress={() => handlePress(id)} />
</View>

// Good: stable references
<View style={styles.container}>
  <Pressable onPress={handlePress} />
</View>
```

## Performance Optimization

**Memoize expensive components:**
```tsx
export const ExpensiveList = React.memo<Props>(({ items, onSelect }) => {
  return (
    <View>
      {items.map(item => (
        <ListItem key={item.id} item={item} onSelect={onSelect} />
      ))}
    </View>
  );
});
```

**FlatList optimization:**
```tsx
<FlatList
  data={items}
  keyExtractor={item => item.id}
  renderItem={renderItem}
  getItemLayout={(_, index) => ({
    length: ITEM_HEIGHT,
    offset: ITEM_HEIGHT * index,
    index,
  })}
  removeClippedSubviews={true}
  maxToRenderPerBatch={10}
  windowSize={5}
  initialNumToRender={10}
/>

// Extract renderItem outside component or memoize
const renderItem = useCallback<ListRenderItem<Item>>(
  ({ item }) => <ItemCard item={item} onPress={handlePress} />,
  [handlePress]
);
```

**useMemo and useCallback correctly:**
```tsx
// Memoize derived data
const sortedItems = useMemo(
  () => items.slice().sort((a, b) => a.name.localeCompare(b.name)),
  [items]
);

// Stable callback references
const handleSubmit = useCallback(async () => {
  await submitForm(formData);
}, [formData]);
```

## Navigation (React Navigation)

**Type-safe navigation:**
```tsx
// types/navigation.ts
export type RootStackParamList = {
  Home: undefined;
  Profile: { userId: string };
  Settings: { section?: string };
};

declare global {
  namespace ReactNavigation {
    interface RootParamList extends RootStackParamList {}
  }
}

// Usage in component
const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
const route = useRoute<RouteProp<RootStackParamList, 'Profile'>>();

navigation.navigate('Profile', { userId: '123' });
```

**Screen options pattern:**
```tsx
const Stack = createNativeStackNavigator<RootStackParamList>();

<Stack.Navigator
  screenOptions={{
    headerShown: false,
    animation: 'slide_from_right',
  }}
>
  <Stack.Screen name="Home" component={HomeScreen} />
  <Stack.Screen
    name="Profile"
    component={ProfileScreen}
    options={{ headerShown: true, title: 'Profile' }}
  />
</Stack.Navigator>
```

## State Management

**Zustand for simple global state:**
```tsx
interface AuthStore {
  user: User | null;
  token: string | null;
  login: (credentials: Credentials) => Promise<void>;
  logout: () => void;
}

export const useAuthStore = create<AuthStore>((set, get) => ({
  user: null,
  token: null,
  login: async (credentials) => {
    const { user, token } = await authApi.login(credentials);
    set({ user, token });
  },
  logout: () => set({ user: null, token: null }),
}));

// Select specific slice to prevent unnecessary re-renders
const user = useAuthStore(state => state.user);
```

**React Query for server state:**
```tsx
const { data, isLoading, error, refetch } = useQuery({
  queryKey: ['user', userId],
  queryFn: () => fetchUser(userId),
  staleTime: 5 * 60 * 1000,
});

const mutation = useMutation({
  mutationFn: updateUser,
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['user'] });
  },
});
```

## Animations (Reanimated)

**Shared values and animated styles:**
```tsx
const offset = useSharedValue(0);

const animatedStyle = useAnimatedStyle(() => ({
  transform: [{ translateX: offset.value }],
}));

const handlePress = () => {
  offset.value = withSpring(offset.value + 50);
};

<Animated.View style={[styles.box, animatedStyle]} />
```

**Gesture handling with Reanimated:**
```tsx
const translateX = useSharedValue(0);

const gesture = Gesture.Pan()
  .onUpdate((e) => {
    translateX.value = e.translationX;
  })
  .onEnd(() => {
    translateX.value = withSpring(0);
  });

<GestureDetector gesture={gesture}>
  <Animated.View style={animatedStyle} />
</GestureDetector>
```

## Styling

**StyleSheet with consistent spacing:**
```tsx
const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 16,
    backgroundColor: colors.background,
  },
  title: {
    fontSize: 24,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 8,
  },
});
```

**Platform-specific styles:**
```tsx
const styles = StyleSheet.create({
  shadow: {
    ...Platform.select({
      ios: {
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.1,
        shadowRadius: 4,
      },
      android: {
        elevation: 4,
      },
    }),
  },
});
```

**Safe area handling:**
```tsx
import { useSafeAreaInsets } from 'react-native-safe-area-context';

const insets = useSafeAreaInsets();

<View style={{ paddingTop: insets.top, paddingBottom: insets.bottom }}>
  {children}
</View>
```

## Error Handling

**Error boundaries for components:**
```tsx
class ErrorBoundary extends Component<Props, State> {
  state = { hasError: false };

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    logError(error, info);
  }

  render() {
    if (this.state.hasError) {
      return <ErrorFallback onRetry={() => this.setState({ hasError: false })} />;
    }
    return this.props.children;
  }
}
```

**API error handling:**
```tsx
try {
  const data = await fetchData();
} catch (error) {
  if (error instanceof ApiError) {
    if (error.status === 401) {
      navigation.navigate('Login');
    } else {
      showToast(error.message);
    }
  } else {
    showToast('Something went wrong');
    logError(error);
  }
}
```

## Testing

**Component testing with RNTL:**
```tsx
import { render, fireEvent, waitFor } from '@testing-library/react-native';

describe('LoginForm', () => {
  it('submits with valid credentials', async () => {
    const onSubmit = jest.fn();
    const { getByPlaceholderText, getByText } = render(
      <LoginForm onSubmit={onSubmit} />
    );

    fireEvent.changeText(getByPlaceholderText('Email'), 'test@example.com');
    fireEvent.changeText(getByPlaceholderText('Password'), 'password123');
    fireEvent.press(getByText('Login'));

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
      });
    });
  });
});
```

## Quality Checklist

Before completing:
- [ ] No inline styles or anonymous functions in render
- [ ] FlatList/ScrollView optimized with proper keys
- [ ] Navigation fully typed
- [ ] Error boundaries wrap critical sections
- [ ] Animations run on UI thread (worklets)
- [ ] Platform-specific code handled correctly
- [ ] Safe areas respected
- [ ] Memory leaks prevented (cleanup in useEffect)
