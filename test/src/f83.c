long f(long n) {
  long sum;
  long i;
  sum = 0;
  for (i = 0; i < n; i = i + 1) {
    if (i % 2 == 0) {
      continue;
    }
    sum = sum + i;
  }
  return sum;
} 