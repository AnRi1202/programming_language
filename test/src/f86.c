long f(long n) {
  long sum;
  long i;
  long j;
  sum = 0;
  
  for (i = 0; i < n; i = i + 1) {
    for (j = 0; j < i; j = j + 1) {
      sum = sum + j;
    }
  }
  
  return sum;
} 