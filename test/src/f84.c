long f(long x) {
  long arr[5];
  long* ptr;
  long i;
  long sum;
  
  for (i = 0; i < 5; i = i + 1) {
    arr[i] = x + i;
  }
  
  ptr = arr;
  sum = 0;
  for (i = 0; i < 5; i = i + 1) {
    sum = sum + *(ptr + i);
  }
  
  return sum;
} 