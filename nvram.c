/*
 * Copyright (c) 2000-2005 Apple Computer, Inc. All rights reserved.
 *
 * @APPLE_LICENSE_HEADER_START@
 *
 * The contents of this file constitute Original Code as defined in and
 * are subject to the Apple Public Source License Version 1.1 (the
 * "License").  You may not use this file except in compliance with the
 * License.  Please obtain a copy of the License at
 * http://www.apple.com/publicsource and read it before using this file.
 *
 * This Original Code and all software distributed under the License are
 * distributed on an "AS IS" basis, WITHOUT WARRANTY OF ANY KIND, EITHER
 * EXPRESS OR IMPLIED, AND APPLE HEREBY DISCLAIMS ALL SUCH WARRANTIES,
 * INCLUDING WITHOUT LIMITATION, ANY WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE OR NON-INFRINGEMENT.  Please see the
 * License for the specific language governing rights and limitations
 * under the License.
 *
 * @APPLE_LICENSE_HEADER_END@
 */
/*
cc -o nvram nvram.c -framework CoreFoundation -framework IOKit -Wall
*/

#include <stdio.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/IOKitKeys.h>
#include <CoreFoundation/CoreFoundation.h>
#include <err.h>
#include <mach/mach_error.h>

// Prototypes
void setup(void);
void teardown(void);
static kern_return_t GetOFVariable(char *name, CFStringRef *nameRef,
				   CFTypeRef *valueRef);
static kern_return_t SetOFVariable(char *name, char *value);
static void DeleteOFVariable(char *name);
void PrintOFVariables(void);
static void PrintOFVariable(const void *key,const void *value,void *context);
static CFTypeRef ConvertValueToCFTypeRef(CFTypeID typeID, char *value);

static void NVRamSyncNow(char *name);

// Global Variables
static io_registry_entry_t gOptionsRef;


void setup()
{
  kern_return_t       result;
  mach_port_t         masterPort;
  
  result = IOMasterPort(bootstrap_port, &masterPort);
  if (result != KERN_SUCCESS) {
    errx(1, "Error getting the IOMaster port: %s",
        mach_error_string(result));
  }
  
  gOptionsRef = IORegistryEntryFromPath(masterPort, "IODeviceTree:/options");
  if (gOptionsRef == 0) {
    errx(1, "nvram is not supported on this system");
  }
}

void teardown()
{
  IOObjectRelease(gOptionsRef);
}


// SetOrGetOFVariable(str)
//
//   Parse the input string, then set or get the specified
//   firmware variable.
//
static void SetOrGetOFVariable(char *str)
{
  long          set = 0;
  char          *name;
  char          *value;
  CFStringRef   nameRef;
  CFTypeRef     valueRef;
  kern_return_t result;
  
  // OF variable name is first.
  name = str;
  
  // Find the equal sign for set
  while (*str) {
    if (*str == '=') {
      set = 1;
      *str++ = '\0';
      break;
    }
    str++;
  }
  
  if (set == 1) {
    // On sets, the OF variable's value follows the equal sign.
    value = str;
    
    result = SetOFVariable(name, value);
	NVRamSyncNow(name);			/* Try syncing the new data to device, best effort! */
    if (result != KERN_SUCCESS) {
      errx(1, "Error setting variable - '%s': %s", name,
           mach_error_string(result));
    }
  } else {
    result = GetOFVariable(name, &nameRef, &valueRef);
    if (result != KERN_SUCCESS) {
      errx(1, "Error getting variable - '%s': %s", name,
           mach_error_string(result));
    }
    
    PrintOFVariable(nameRef, valueRef, 0);
    CFRelease(nameRef);
    CFRelease(valueRef);
  }
}


// GetOFVariable(name, nameRef, valueRef)
//
//   Get the named firmware variable.
//   Return it and it's symbol in valueRef and nameRef.
//
static kern_return_t GetOFVariable(char *name, CFStringRef *nameRef,
				   CFTypeRef *valueRef)
{
  *nameRef = CFStringCreateWithCString(kCFAllocatorDefault, name,
				       kCFStringEncodingUTF8);
  if (*nameRef == 0) {
    errx(1, "Error creating CFString for key %s", name);
  }
  
  *valueRef = IORegistryEntryCreateCFProperty(gOptionsRef, *nameRef, 0, 0);
  if (*valueRef == 0) return kIOReturnNotFound;
  
  return KERN_SUCCESS;
}


// SetOFVariable(name, value)
//
//   Set or create an firmware variable with name and value.
//
static kern_return_t SetOFVariable(char *name, char *value)
{
  CFStringRef   nameRef;
  CFTypeRef     valueRef;
  CFTypeID      typeID;
  kern_return_t result = KERN_SUCCESS;
  
  nameRef = CFStringCreateWithCString(kCFAllocatorDefault, name,
				      kCFStringEncodingUTF8);
  if (nameRef == 0) {
    errx(1, "Error creating CFString for key %s", name);
  }
  
  valueRef = IORegistryEntryCreateCFProperty(gOptionsRef, nameRef, 0, 0);
  if (valueRef) {
    typeID = CFGetTypeID(valueRef);
    CFRelease(valueRef);
    
    valueRef = ConvertValueToCFTypeRef(typeID, value);
    if (valueRef == 0) {
      errx(1, "Error creating CFTypeRef for value %s", value);
    }  result = IORegistryEntrySetCFProperty(gOptionsRef, nameRef, valueRef);
  } else {
    while (1) {
      // In the default case, try data, string, number, then boolean.    
      
      valueRef = ConvertValueToCFTypeRef(CFDataGetTypeID(), value);
      if (valueRef != 0) {
	result = IORegistryEntrySetCFProperty(gOptionsRef, nameRef, valueRef);
	if (result == KERN_SUCCESS) break;
      }
      
      valueRef = ConvertValueToCFTypeRef(CFStringGetTypeID(), value);
      if (valueRef != 0) {
	result = IORegistryEntrySetCFProperty(gOptionsRef, nameRef, valueRef);
	if (result == KERN_SUCCESS) break;
      }
      
      valueRef = ConvertValueToCFTypeRef(CFNumberGetTypeID(), value);
      if (valueRef != 0) {
	result = IORegistryEntrySetCFProperty(gOptionsRef, nameRef, valueRef);
	if (result == KERN_SUCCESS) break;
      }
      
      valueRef = ConvertValueToCFTypeRef(CFBooleanGetTypeID(), value);
      if (valueRef != 0) {
	result = IORegistryEntrySetCFProperty(gOptionsRef, nameRef, valueRef);
	if (result == KERN_SUCCESS) break;
      }
      
      break;
    }
  }
  
  CFRelease(nameRef);
  
  return result;
}


// DeleteOFVariable(name)
//
//   Delete the named firmware variable.
//   
//
static void DeleteOFVariable(char *name)
{
  SetOFVariable(kIONVRAMDeletePropertyKey, name);
}

static void NVRamSyncNow(char *name)
{
  SetOFVariable(kIONVRAMSyncNowPropertyKey, name);
}

// PrintOFVariables()
//
//   Print all of the firmware variables.
//
void PrintOFVariables()
{
  kern_return_t          result;
  CFMutableDictionaryRef dict;
  
  result = IORegistryEntryCreateCFProperties(gOptionsRef, &dict, 0, 0);
  if (result != KERN_SUCCESS) {
    errx(1, "Error getting the firmware variables: %s", mach_error_string(result));
  }

  CFDictionaryApplyFunction(dict, &PrintOFVariable, 0);
  
  CFRelease(dict);
}

// PrintOFVariable(key, value, context)
//
//   Print the given firmware variable.
//
static void PrintOFVariable(const void *key, const void *value, void *context)
{
  long          cnt, cnt2;
  CFIndex       nameLen;
  char          *nameBuffer = 0;
  const char    *nameString;
  char          numberBuffer[10];
  const uint8_t *dataPtr;
  uint8_t       dataChar;
  char          *dataBuffer = 0;
  CFIndex       valueLen;
  char          *valueBuffer = 0;
  const char    *valueString = 0;
  uint32_t      number, length;
  CFTypeID      typeID;
  
  // Get the OF variable's name.
  nameLen = CFStringGetLength(key) + 1;
  nameBuffer = malloc(nameLen);
  if( nameBuffer && CFStringGetCString(key, nameBuffer, nameLen, kCFStringEncodingUTF8) )
    nameString = nameBuffer;
  else {
    warnx("Unable to convert property name to C string");
    nameString = "<UNPRINTABLE>";
  }
  
  // Get the OF variable's type.
  typeID = CFGetTypeID(value);
  
  if (typeID == CFBooleanGetTypeID()) {
    if (CFBooleanGetValue(value)) valueString = "true";
    else valueString = "false";
  } else if (typeID == CFNumberGetTypeID()) {
    CFNumberGetValue(value, kCFNumberSInt32Type, &number);
    if (number == 0xFFFFFFFF) sprintf(numberBuffer, "-1");
    else if (number < 1000) sprintf(numberBuffer, "%d", number);
    else sprintf(numberBuffer, "0x%x", number);
    valueString = numberBuffer;
  } else if (typeID == CFStringGetTypeID()) {
    valueLen = CFStringGetLength(value) + 1;
    valueBuffer = malloc(valueLen + 1);
    if ( valueBuffer && CFStringGetCString(value, valueBuffer, valueLen, kCFStringEncodingUTF8) )
      valueString = valueBuffer;
    else {
      warnx("Unable to convert value to C string");
      valueString = "<UNPRINTABLE>";
    }
  } else if (typeID == CFDataGetTypeID()) {
    length = CFDataGetLength(value);
    if (length == 0) valueString = "";
    else {
      dataBuffer = malloc(length * 3 + 1);
      if (dataBuffer != 0) {
	dataPtr = CFDataGetBytePtr(value);
	for (cnt = cnt2 = 0; cnt < length; cnt++) {
	  dataChar = dataPtr[cnt];
	  if (isprint(dataChar)) dataBuffer[cnt2++] = dataChar;
	  else {
	    sprintf(dataBuffer + cnt2, "%%%02x", dataChar);
	    cnt2 += 3;
	  }
	}
	dataBuffer[cnt2] = '\0';
	valueString = dataBuffer;
      }
    }
  } else {
    valueString="<INVALID>";
  }
  
  if ((nameString != 0) && (valueString != 0))
    printf("%s\t%s\n", nameString, valueString);
  
  if (dataBuffer != 0) free(dataBuffer);
  if (nameBuffer != 0) free(nameBuffer);
  if (valueBuffer != 0) free(valueBuffer);
}


// ConvertValueToCFTypeRef(typeID, value)
//
//   Convert the value into a CFType given the typeID.
//
static CFTypeRef ConvertValueToCFTypeRef(CFTypeID typeID, char *value)
{
  CFTypeRef     valueRef = 0;
  long          cnt, cnt2, length;
  unsigned long number, tmp;
  
  if (typeID == CFBooleanGetTypeID()) {
    if (!strcmp("true", value)) valueRef = kCFBooleanTrue;
    else if (!strcmp("false", value)) valueRef = kCFBooleanFalse;
  } else if (typeID == CFNumberGetTypeID()) {
    number = strtol(value, 0, 0);
    valueRef = CFNumberCreate(kCFAllocatorDefault, kCFNumberSInt32Type,
			      &number);
  } else if (typeID == CFStringGetTypeID()) {
    valueRef = CFStringCreateWithCString(kCFAllocatorDefault, value,
					 kCFStringEncodingUTF8);
  } else if (typeID == CFDataGetTypeID()) {
    length = strlen(value);
    for (cnt = cnt2 = 0; cnt < length; cnt++, cnt2++) {
      if (value[cnt] == '%') {
	if (!ishexnumber(value[cnt + 1]) ||
	    !ishexnumber(value[cnt + 2])) return 0;
	number = toupper(value[++cnt]) - '0';
	if (number > 9) number -= 7;
	tmp = toupper(value[++cnt]) - '0';
	if (tmp > 9) tmp -= 7;
	number = (number << 4) + tmp;
	value[cnt2] = number;
      } else value[cnt2] = value[cnt];
    }
    valueRef = CFDataCreateWithBytesNoCopy(kCFAllocatorDefault, (const UInt8 *)value,
					   cnt2, kCFAllocatorDefault);
  } else return 0;
  
  return valueRef;
}
